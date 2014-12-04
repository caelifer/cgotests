package main

/*
#include <dirent.h>
#include <stdlib.h>
#include "walk.h"

extern void printNode(const char *, struct dirent *);
extern int NodeCounter;
extern int DirCounter;

void doWalkNode(const char *path, struct dirent *node) {
	WalkNode(path, node, printNode);
}
*/
import "C"

import (
	"fmt"
	"os"
	"unsafe"
)

func main() {
	paths := os.Args[1:]

	if len(paths) == 0 {
		paths = []string{"."}
	}

	Walk(paths, nil, nil)
	// Print stats
	fmt.Fprintf(os.Stderr, "\nTotal: %d nodes, %d directories, %d otheres\n",
		int(C.NodeCounter), int(C.DirCounter), int(C.NodeCounter)-int(C.DirCounter),
	)
}

func Walk(paths []string, node Node, fn NodeFn) {
	for _, p := range paths {
		cpath := C.CString(p)
		defer C.free(unsafe.Pointer(cpath))

		// 		// Wrap our client call back
		// 		callback := func(p *C.char, dirent *C.struct_dirent) {
		// 			// Convert back to Go
		// 			node := MakeNodeFromDirent(dirent)
		// 			path := C.GoString(p)
		//
		// 			// Call client
		// 			fn(path, node)
		// 		}

		// C.WalkNode(cpath, nil, (*C.func_CallBack)(unsafe.Pointer(&callback)))
		// C.WalkNode(cpath, nil, *(**[0]byte)(unsafe.Pointer(&callback)))
		C.doWalkNode(cpath, nil)
	}
}

func MakeNodeFromDirent(dirent *C.struct_dirent) Node {
	node := &node_t{}

	node.name = C.GoString((*C.char)(&dirent.d_name[0]))

	switch C.uchar(dirent.d_type) {
	case C.DT_BLK:
		node.kind = NTBlockDevice
	case C.DT_CHR:
		node.kind = NTCharDevice
	case C.DT_DIR:
		node.kind = NTDirectory
	case C.DT_FIFO:
		node.kind = NTFIFO
	case C.DT_LNK:
		node.kind = NTSymLink
	case C.DT_REG:
		node.kind = NTRegular
	case C.DT_SOCK:
		node.kind = NTSocket
	default:
		node.kind = NTUnknown
	}

	return node
}

type NodeFn func(path string, node Node)

type NodeType uint8

const (
	NTBlockDevice NodeType = iota // 	DT_BLK      This is a block device.
	NTCharDevice                  // 	DT_CHR      This is a character device.
	NTDirectory                   // 	DT_DIR      This is a directory.
	NTFIFO                        // 	DT_FIFO     This is a named pipe (FIFO).
	NTSymLink                     // 	DT_LNK      This is a symbolic link.
	NTRegular                     // 	DT_REG      This is a regular file.
	NTSocket                      // 	DT_SOCK     This is a UNIX domain socket.
	NTUnknown                     // 	DT_UNKNOWN  The file type is unknown.
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
	default:
		return "UKN"
	}
}

type Node interface {
	Name() string   // Node's short name
	Type() NodeType // Node's type
}

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

// func printNode(path string, node Node) {
// 	fmt.Printf("[%s] %s\n", node.Type(), path)
// }
