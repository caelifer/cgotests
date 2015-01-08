#include <dirent.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>

#include <sys/types.h>
#include <sys/stat.h>
#include <unistd.h>

#include <err.h>

#include "walk.h"

#ifndef NAME_MAX
#define NAME_MAX 1024
#endif

int NodeCounter = 0;
int DirCounter = 0;

// Disable main compiliation when building as a library
#ifdef XXX_MAIN_ENABLED_XXX

// Forward declaration
static struct dirent *createNode(const char *path);
static int dots(const char *name);
static char *createNewPath(const char *oldp, char *newp);

/*
* Implementation
*/

static char *createNewPath(const char *oldp, char *newp) {
	// Construct path
	int sz = strlen(oldp) + strlen(newp) + 2; // +2 for last '0' and middle '/'
	char *newPath = malloc(sizeof(char) * sz);

	if (newPath == NULL) {
		perror("memory [WT]");
		return NULL;
	}

	// Finally build new path
	sprintf(newPath, "%s/%s", strcmp(oldp, "/") == 0 ? "" : oldp, newp);
	return newPath;
}

// WalkTree
void WalkTree(const char *path, DIR *dir, CallBack cb) {
	struct dirent node, *result;

	// Read all entries one by one
	for (result = &node; readdir_r(dir, &node, &result) == 0 && result != NULL;) {
		// Skip . && ..
		if (dots(node.d_name)) {
			continue;
		}

        // Build new path
		char *newPath = createNewPath(path, node.d_name);

		// Walk each node (recursively)
		WalkNode(newPath, &node, cb);

		// Make sure to free allocated newPath
		free(newPath);
	}

	// Process errors
	if (result == NULL) {
		// EOF - done
		return;
	} else {
		perror(path);
	}
}

// WalkNode
void WalkNode(const char *path, struct dirent *node, CallBack cb) {
	int needFree = 0;

	NodeCounter++; // Increment node count

	// If node is NULL, populate node
	// If node is DT_UNKNOWN - re-populate (wierd bug that is cured by calling lstat(2))
	if (node == NULL || node->d_type == DT_UNKNOWN) {
		if ((node = createNode(path)) == NULL) {
			perror("node");
			return;
		}
		needFree = 1;
	}

	// At this point our data structures are fully populated
	
	// First run call back
	cb(path, node);

	// Check if node is directory and call WalkTree on it
	if (node->d_type == DT_DIR) {

		DirCounter++; // Increment directory count

		DIR *dir = opendir(path);

		if (dir == NULL) {
			// perror("opendir");
			warn("'%s'", path);
			return;
		}

		// Recurse via WalkTree call
		WalkTree(path, dir, cb);

		// Always close open directory
		closedir(dir);
	}

	if (needFree) {
		free(node); // Free node if we created it
	}
}

// Simple function to check if names are '.' or '..'
static int dots(const char *name) {
	return strncmp(name, "..", NAME_MAX) == 0 || strncmp(name, ".", NAME_MAX) == 0;
}

// Dirent node constructor
static struct dirent *createNode(const char *path) {
		struct dirent *node;
		struct stat buf;

		// Populate node by doing lstat(2)
		node = malloc(sizeof(struct dirent));
		if (node == NULL) {
			perror("memory [WN]");
			return NULL;
		}

		// Get stats
		if (lstat(path, &buf) == -1) {
            fprintf(stderr, "Failed to lstat '%s'\n", path);
			perror("lstat");
			return NULL;
		}

		// Set node attributes
		char *p = rindex(path, '/');
		strncpy(node->d_name, (p == NULL) ? path : p+1, NAME_MAX); // empty string is a name for system root

		// Set inode
		node->d_ino = buf.st_ino;

		// Set type
		if (S_ISREG(buf.st_mode)) {
			node->d_type = DT_REG;
		} else if (S_ISDIR(buf.st_mode)) {
			node->d_type = DT_DIR;
		} else if (S_ISCHR(buf.st_mode)) {
			node->d_type = DT_CHR;
		} else if (S_ISBLK(buf.st_mode)) {
			node->d_type = DT_BLK;
		} else if (S_ISFIFO(buf.st_mode)) {
			node->d_type = DT_FIFO;
		} else if (S_ISLNK(buf.st_mode)) {
			node->d_type = DT_LNK;
		} else if (S_ISSOCK(buf.st_mode)) {
			node->d_type = DT_SOCK;
		} else {
			node->d_type = DT_UNKNOWN;
		}

		return node;
}

void printNode(const char *p, struct dirent *de);

int main(int argc, char *argv[]) {
	int i;

	if (argc == 1) {
		WalkNode(".", NULL, printNode);
	} else {
		for (i = argc - 1; i > 0; i--) {
			WalkNode(*(++argv), NULL, printNode);
		}
	}

	fprintf(stderr, "\nTotal: %d nodes, %d directories, %d otheres\n", NodeCounter, DirCounter, NodeCounter - DirCounter);
}

/* CallBack implementation */ 
void printNode(const char *path, struct dirent *de) {
	char *type;

	switch (de->d_type) {
		case DT_REG:
			type = "REG"; break;
		case DT_DIR:
			type = "DIR"; break;
		case DT_CHR:
			type = "CHR"; break;
		case DT_BLK:
			type = "BLK"; break;
		case DT_FIFO:
			type = "FIO"; break;
		case DT_LNK:
			type = "LNK"; break;
		case DT_SOCK:
			type = "SCK"; break;
		default:
			type = "UNK";
	}

	printf("[%s] %s\n", type, path);
}

#endif /* XXX_MAIN_ENABLED_XXX */
/*
 * vim: :ts=4:sw=4:noexpandtab:nohls:ai:
 */
