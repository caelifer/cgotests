#include <dirent.h>

extern void printNode(const char* p, struct dirent *de);

void myPrint(char *p, struct dirent *de) {
	printNode(p, de);
}
