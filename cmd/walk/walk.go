package main

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sync/atomic"
	"unsafe"
)

/*
#include <dirent.h>
#include <stdlib.h>
*/
import "C"

type CustomFn func(path string, node Node) error

var (
	ProgName                string // Progname
	NodeCounter, DirCounter uint64 // Counters
)

func init() {
	// Get program name
	ProgName = path.Base(os.Args[0])
}

func main() {
	paths := os.Args[1:]

	if len(paths) == 0 {
		paths = []string{"."}
	}

	// Walk provided path with the custom function
	Walk(paths, nil, func(path string, node Node) error {
		fmt.Printf("[%s] %s\n", node.Type(), path)
		return nil
	})

	// Print stats
	fmt.Fprintf(os.Stderr, "\nTotal: %d nodes, %d directories, %d otheres\n",
		NodeCounter, DirCounter, NodeCounter-DirCounter,
	)
}

func Walk(paths []string, node Node, fn CustomFn) {
	for _, p := range paths {
		if err := walkNode(p, node, fn); err != nil {
			warning(p, err)
		}
	}
}

func warning(path string, err error) {
	fmt.Fprintf(os.Stderr, "%s: %q: %s\n", ProgName, path, err)
}

func walkNode(path string, node Node, fn CustomFn) error {
	var err error

	// increment node counter
	atomic.AddUint64(&NodeCounter, 1)

	// Construct new node from path if not provided
	if node == nil {
		if node, err = createNode(path); err != nil {
			// Report an error if we can't create a node from the path
			return err
		}
	}

	// Increment directory count if node is a directory
	if node.Type() == NTDirectory {
		atomic.AddUint64(&DirCounter, 1)
	}

	// Call CustomFn
	if err = fn(path, node); err != nil {
		return err
	}

	// Recursevly process directory
	if node.Type() == NTDirectory {
		err = walkDir(path, node, fn)
	}

	return err
}

func walkDir(path string, node Node, fn CustomFn) error {
	var (
		err    error
		de     C.struct_dirent
		result *C.struct_dirent
	)

	// Convert path to C-string
	cpath := C.CString(path)
	defer C.free(unsafe.Pointer(cpath))

	dir, err := C.opendir(cpath)
	if err != nil {
		return err
	}

	// Node accumulator
	nodes := make([]Node, 0, 1024)
	for result = &de; C.readdir_r((*C.DIR)(dir), &de, &result) == 0 && result != nil; {
		if dotDirs(getNameFromDirent(&de)) {
			// skip '.' and '..'
			continue
		}
		nodes = append(nodes, createNodeFromDirent(path, &de))
	}
	// Close directory ASAP
	C.closedir(dir)

	// Process all accumulated nodes if no errors were detected
	if result == nil {
		for _, node := range nodes {
			newPath := filepath.Join(path, node.Name())
			// Walk each node
			if err = walkNode(newPath, node, fn); err != nil {
				warning(newPath, err)
			}
		}
		return nil
	}

	panic("should never reach here")
}

func dotDirs(name string) bool {
	return name == "." || name == ".."
}

func createNode(path string) (Node, error) {
	fi, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	return makeNodeFromFileInfo(fi), nil
}

func makeNodeFromFileInfo(fi os.FileInfo) Node {
	return &node_t{
		name: fi.Name(),
		kind: castFileModeToNodeType(fi.Mode()),
	}
}

func castFileModeToNodeType(fm os.FileMode) NodeType {
	switch fm & os.ModeType {
	case os.ModeDir:
		return NTDirectory
	case os.ModeDevice:
		return NTBlockDevice
	case os.ModeCharDevice:
		return NTCharDevice
	case os.ModeSymlink:
		return NTSymLink
	case os.ModeSocket:
		return NTSocket
	case os.ModeNamedPipe:
		return NTFIFO
	}

	if fm.IsRegular() {
		return NTRegular
	}

	return NTUnknown
}

func getNameFromDirent(dirent *C.struct_dirent) string {
	return C.GoString((*C.char)(&dirent.d_name[0]))
}

func createNodeFromDirent(path string, dirent *C.struct_dirent) Node {
	node := &node_t{
		name: getNameFromDirent(dirent),
	}

	switch C.uchar(dirent.d_type) {
	case C.DT_FIFO:
		node.kind = NTFIFO
	case C.DT_CHR:
		node.kind = NTCharDevice
	case C.DT_DIR:
		node.kind = NTDirectory
	case C.DT_BLK:
		node.kind = NTBlockDevice
	case C.DT_REG:
		node.kind = NTRegular
	case C.DT_LNK:
		node.kind = NTSymLink
	case C.DT_SOCK:
		node.kind = NTSocket
	case C.DT_WHT:
		node.kind = NTWhiteout
	default:
		// Try using lstat(2) [fixing readdir_r(2) bug]
		fi, err := os.Lstat(filepath.Join(path, node.Name()))
		if err == nil {
			node.kind = castFileModeToNodeType(fi.Mode())
		} else {
			node.kind = NTUnknown
		}
	}
	return node
}

type NodeType uint8

const (
	NTBlockDevice NodeType = iota // DT_BLK      This is a block device.
	NTCharDevice                  // DT_CHR      This is a character device.
	NTDirectory                   // DT_DIR      This is a directory.
	NTFIFO                        // DT_FIFO     This is a named pipe (FIFO).
	NTSymLink                     // DT_LNK      This is a symbolic link.
	NTRegular                     // DT_REG      This is a regular file.
	NTSocket                      // DT_SOCK     This is a UNIX domain socket.
	NTWhiteout                    // DT_WHT      This is BSD-style whiteout
	NTUnknown                     // DT_UNKNOWN  The file type is unknown.
)

func (nt NodeType) String() string {
	switch nt {
	case NTBlockDevice:
		return "BLK"
	case NTCharDevice:
		return "CHR"
	case NTDirectory:
		return "DIR"
	case NTFIFO:
		return "FIO"
	case NTSymLink:
		return "LNK"
	case NTRegular:
		return "REG"
	case NTSocket:
		return "SCK"
	case NTWhiteout:
		return "WHT"
	default:
		return "UKN"
	}
}

type Node interface {
	Name() string   // Node's short name
	Type() NodeType // Node's type
}

// Concreate implementation of Node interface
type node_t struct {
	name string
	kind NodeType
}

func (n node_t) Name() string {
	return n.name
}

func (n node_t) Type() NodeType {
	return n.kind
}

// vim: :ts=4:sw=4:noexpandtab:nohls:ai:
