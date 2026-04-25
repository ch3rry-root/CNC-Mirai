#ifndef SYSINFO_H
#define SYSINFO_H

#include <stdint.h>

// Devuelve la RAM total en MB (0 si error)
uint32_t get_total_ram_mb(void);

// Devuelve el número de núcleos de CPU (mínimo 1)
uint8_t get_cpu_cores(void);

#endif
