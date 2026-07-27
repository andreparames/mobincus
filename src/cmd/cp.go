package cmd

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"mobincus/incus"

	"github.com/spf13/cobra"
)

var cpFollowLink bool

var cpCmd = &cobra.Command{
	Use:   "cp [OPTIONS] CONTAINER:SRC_PATH DEST_PATH|-",
	Short: "Copy files/folders between a container and the local filesystem",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		src := args[0]
		dst := args[1]

		srcContainer, srcPath := splitCpRef(src)
		dstContainer, dstPath := splitCpRef(dst)

		client := incus.NewClient()

		switch {
		case srcContainer != "" && dstContainer != "":
			return fmt.Errorf("copying between containers is not supported")
		case srcContainer != "":
			return copyFromContainer(client, srcContainer, srcPath, dst)
		case dstContainer != "":
			return copyToContainer(client, dstContainer, dstPath, src)
		default:
			return fmt.Errorf("must specify at least one container source")
		}
	},
}

func splitCpRef(ref string) (container, path string) {
	if !strings.Contains(ref, ":") {
		return "", ref
	}
	idx := strings.Index(ref, ":")
	possibleContainer := ref[:idx]
	if possibleContainer == "" || possibleContainer[0] == '/' || possibleContainer[0] == '~' || possibleContainer[0] == '.' {
		return "", ref
	}
	return possibleContainer, ref[idx+1:]
}

func copyFromContainer(client *incus.Client, container, srcPath, dstPath string) error {
	info, err := client.FileStat(container, srcPath)
	if err != nil {
		return fmt.Errorf("error: no such object: %s", srcPath)
	}

	if dstPath == "-" {
		return streamTarToStdout(client, container, srcPath, info)
	}

	absDst, err := filepath.Abs(dstPath)
	if err != nil {
		return err
	}

	if info.Type == "directory" {
		return copyDirFromContainer(client, container, srcPath, absDst)
	}

	return copyFileFromContainer(client, container, srcPath, absDst, info)
}

func copyFileFromContainer(client *incus.Client, container, srcPath, dstPath string, info *incus.FileInfo) error {
	body, _, err := client.FileGet(container, srcPath)
	if err != nil {
		return err
	}
	defer body.Close()

	f, err := os.Create(dstPath)
	if err != nil {
		return err
	}
	defer f.Close()

	if info.Mode != 0 {
		f.Chmod(info.Mode)
	}

	_, err = io.Copy(f, body)
	return err
}

func copyDirFromContainer(client *incus.Client, container, srcPath, dstPath string) error {
	if err := os.MkdirAll(dstPath, 0755); err != nil {
		return err
	}

	entries, err := client.FileList(container, srcPath)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		childSrc := srcPath + "/" + entry
		childDst := filepath.Join(dstPath, entry)

		childInfo, err := client.FileStat(container, childSrc)
		if err != nil {
			return err
		}

		if childInfo.Type == "directory" {
			if err := copyDirFromContainer(client, container, childSrc, childDst); err != nil {
				return err
			}
		} else {
			if err := copyFileFromContainer(client, container, childSrc, childDst, childInfo); err != nil {
				return err
			}
		}
	}

	return nil
}

func streamTarToStdout(client *incus.Client, container, srcPath string, info *incus.FileInfo) error {
	gw := gzip.NewWriter(os.Stdout)
	defer gw.Close()

	tw := tar.NewWriter(gw)
	defer tw.Close()

	return addToTar(client, container, srcPath, info, tw)
}

func addToTar(client *incus.Client, container, path string, info *incus.FileInfo, tw *tar.Writer) error {
	if info.Type == "directory" {
		hdr := &tar.Header{
			Name:     filepath.Base(path),
			Mode:     0755,
			Typeflag: tar.TypeDir,
		}
		if err := tw.WriteHeader(hdr); err != nil {
			return err
		}

		entries, err := client.FileList(container, path)
		if err != nil {
			return err
		}
		for _, entry := range entries {
			childPath := path + "/" + entry
			childInfo, err := client.FileStat(container, childPath)
			if err != nil {
				return err
			}
			if err := addToTar(client, container, childPath, childInfo, tw); err != nil {
				return err
			}
		}
		return nil
	}

	body, fileInfo, err := client.FileGet(container, path)
	if err != nil {
		return err
	}
	defer body.Close()

	hdr := &tar.Header{
		Name:     filepath.Base(path),
		Size:     fileInfo.Size,
		Mode:     int64(fileInfo.Mode),
		Typeflag: tar.TypeReg,
	}
	if err := tw.WriteHeader(hdr); err != nil {
		return err
	}
	_, err = io.Copy(tw, body)
	return err
}

func copyToContainer(client *incus.Client, container, dstPath, srcPath string) error {
	srcInfo, err := os.Stat(srcPath)
	if err != nil {
		return err
	}

	if srcInfo.IsDir() {
		return copyDirToContainer(client, container, dstPath, srcPath)
	}

	return copyFileToContainer(client, container, dstPath, srcPath)
}

func copyFileToContainer(client *incus.Client, container, dstPath, srcPath string) error {
	f, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return err
	}

	return client.FilePut(container, dstPath, "file", info.Mode(), f)
}

func copyDirToContainer(client *incus.Client, container, dstPath, srcPath string) error {
	if err := client.FilePut(container, dstPath, "directory", 0755, nil); err != nil {
		return err
	}

	return filepath.Walk(srcPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(srcPath, path)
		dest := dstPath + "/" + rel

		if info.IsDir() {
			return client.FilePut(container, dest, "directory", info.Mode(), nil)
		}

		return copyFileToContainer(client, container, dest, path)
	})
}

func init() {
	cpCmd.Flags().BoolVarP(&cpFollowLink, "follow-link", "L", false, "Always follow symbol link in SRC_PATH")
	rootCmd.AddCommand(cpCmd)
}
