package main

import (
	"archive/tar"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

// secureJoin resolves rel inside root the way the container will see it: symlinks are followed,
// but absolute targets and ".." can never leave root. Only the parent path is resolved, so the
// last element may itself be a symlink (it gets replaced, not followed).
func secureJoin(root, rel string) (string, error) {
	rel = path.Clean("/" + rel)
	if rel == "/" {
		return root, nil
	}
	dir, base := path.Split(rel)
	parts := strings.Split(strings.Trim(dir, "/"), "/")
	cur := "/"
	for hops := 0; len(parts) > 0; {
		p := parts[0]
		parts = parts[1:]
		if p == "" || p == "." {
			continue
		}
		if p == ".." {
			cur = path.Dir(cur)
			continue
		}
		next := path.Join(cur, p)
		fi, err := os.Lstat(filepath.Join(root, next))
		if err == nil && fi.Mode()&os.ModeSymlink != 0 {
			if hops++; hops > 255 {
				return "", errors.New("too many symlinks")
			}
			t, err := os.Readlink(filepath.Join(root, next))
			if err != nil {
				return "", err
			}
			if path.IsAbs(t) {
				cur = "/"
			}
			parts = append(strings.Split(strings.Trim(t, "/"), "/"), parts...)
			continue
		}
		cur = next
	}
	return filepath.Join(root, cur, base), nil
}

// applyLayer extracts one gzip/tar layer onto rootfs, honouring OCI whiteouts
func applyLayer(rootfs, file string) error {
	f, err := os.Open(file)
	if err != nil {
		return err
	}
	defer f.Close()
	var r io.Reader = f
	magic := make([]byte, 2)
	if _, err := io.ReadFull(f, magic); err != nil {
		return err
	}
	f.Seek(0, 0)
	if magic[0] == 0x1f && magic[1] == 0x8b {
		gz, err := gzip.NewReader(f)
		if err != nil {
			return err
		}
		defer gz.Close()
		r = gz
	} else if magic[0] == 0x28 && magic[1] == 0xb5 {
		return errors.New("zstd layers are not supported")
	}

	tr := tar.NewReader(r)
	written := map[string]bool{} // paths created by this layer (an opaque whiteout only hides lower layers)
	type dirTime struct {
		p string
		t time.Time
	}
	var dirs []dirTime
	for {
		h, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		name := path.Clean("/" + h.Name)
		dir, base := path.Split(name)

		if base == ".wh..wh..opq" { // opaque dir: drop everything from lower layers
			d, err := secureJoin(rootfs, dir)
			if err != nil {
				return err
			}
			ents, _ := os.ReadDir(d)
			for _, e := range ents {
				if !written[path.Join(dir, e.Name())] {
					os.RemoveAll(filepath.Join(d, e.Name()))
				}
			}
			continue
		}
		if strings.HasPrefix(base, ".wh.") {
			t, err := secureJoin(rootfs, path.Join(dir, strings.TrimPrefix(base, ".wh.")))
			if err != nil {
				return err
			}
			os.RemoveAll(t)
			continue
		}

		dst, err := secureJoin(rootfs, name)
		if err != nil {
			return err
		}
		if dst == rootfs {
			continue
		}
		written[name] = true
		os.MkdirAll(filepath.Dir(dst), 0o755)
		mode := os.FileMode(h.Mode & 0o7777)
		if h.Typeflag != tar.TypeDir {
			if _, err := os.Lstat(dst); err == nil {
				os.RemoveAll(dst) // a later layer replaces whatever was there
			}
		}
		switch h.Typeflag {
		case tar.TypeDir:
			if fi, err := os.Lstat(dst); err == nil && !fi.IsDir() {
				os.Remove(dst)
			}
			if err := os.MkdirAll(dst, 0o755); err != nil {
				return err
			}
			dirs = append(dirs, dirTime{dst, h.ModTime})
		case tar.TypeReg, tar.TypeRegA:
			out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
			if err != nil {
				return err
			}
			if _, err := io.Copy(out, tr); err != nil {
				out.Close()
				return err
			}
			out.Close()
		case tar.TypeSymlink:
			if err := os.Symlink(h.Linkname, dst); err != nil {
				return err
			}
		case tar.TypeLink:
			src, err := secureJoin(rootfs, h.Linkname)
			if err != nil {
				return err
			}
			if err := os.Link(src, dst); err != nil {
				return fmt.Errorf("hardlink %s: %w", name, err)
			}
			continue // shares the inode, attributes come from the target
		case tar.TypeChar, tar.TypeBlock, tar.TypeFifo:
			m := uint32(syscall.S_IFIFO)
			if h.Typeflag == tar.TypeChar {
				m = syscall.S_IFCHR
			} else if h.Typeflag == tar.TypeBlock {
				m = syscall.S_IFBLK
			}
			// device nodes are optional: nspawn provides /dev itself
			_ = syscall.Mknod(dst, m|uint32(mode), int(h.Devmajor<<8|h.Devminor))
		default:
			continue
		}
		_ = os.Lchown(dst, h.Uid, h.Gid)
		if h.Typeflag != tar.TypeSymlink {
			_ = os.Chmod(dst, mode) // after chown: chown clears setuid bits
			_ = os.Chtimes(dst, h.ModTime, h.ModTime)
		}
	}
	for i := len(dirs) - 1; i >= 0; i-- {
		_ = os.Chtimes(dirs[i].p, dirs[i].t, dirs[i].t)
	}
	return nil
}
