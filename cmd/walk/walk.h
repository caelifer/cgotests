#ifndef _FIND_H
#define _FIND_H

#include <dirent.h>

/*
* Prototypes
*/

typedef void (*CallBack)(const char *path, struct dirent *node);

void WalkTree(const char* path, DIR *dir, CallBack cb);

void WalkNode(const char *path, struct dirent *node, CallBack cb);

void printNode(const char *path, struct dirent *node);


#endif /* _FIND_H */
