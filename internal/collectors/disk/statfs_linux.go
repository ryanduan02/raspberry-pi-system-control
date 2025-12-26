package disk

import "golang.org/x/sys/unix"

func statfs(path string) (*unix.Statfs_t, error) {
	var st unix.Statfs_t
	if err := unix.Statfs(path, &st); err != nil {
		return nil, err
	}
	return &st, nil
}