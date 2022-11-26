package nsenter

/*
#define _GNU_SOURCE

#include <unistd.h>
#include <errno.h>
#include <sched.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <fcntl.h>

__attribute__((constructor)) static void enter_namespace() {
    char *my_docker_pid;
    my_docker_pid = getenv("my_docker_pid");
    if (my_docker_pid) {
        fprintf(stdout, "got my_docker_pid=%s\n", my_docker_pid);
    } else {
        fprintf(stdout, "missing my_docker_pid env skip nsenter");
        return;
    }

    char *my_docker_cmd;
    my_docker_cmd = getenv("my_docker_cmd");
    if (my_docker_cmd) {
        fprintf(stdout, "got my_docker_cmd=%s\n", my_docker_cmd);
    } else {
        fprintf(stdout, "missing my_docker_cmd env skip nsenter");
        return;
    }

    int i;
    char nsPath[1024];
    char *namespaces[] = {"ipc", "uts", "net", "pid", "mnt"};

    for (i = 0; i < 5; i++) {
        sprintf(nsPath, "/proc/%s/ns/%s", my_docker_pid, namespaces[i]);
        fprintf(stdout, "try to join namespace %s\n", nsPath);
        int fd = open(nsPath, O_RDONLY);
        if (setns(fd, 0) == -1) {
            fprintf(stderr, "setns on %s namespace failed: %s\n", namespaces[i], strerror(errno));
        } else {
            fprintf(stdout, "setns on %s namespace succeeded\n", namespaces[i]);
        }
        close(fd);
    }

    int res = system(my_docker_cmd);
    exit(0);
    return;
}
*/
import "C"
