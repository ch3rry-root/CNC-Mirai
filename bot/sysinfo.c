#include "sysinfo.h"
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <unistd.h>
#include <stdint.h>

static uint32_t get_total_ram_mb_fallback(void) {
    FILE *fp = fopen("/proc/meminfo", "r");
    if (!fp) return 0;
    uint32_t total_kb = 0;
    char line[256];
    while (fgets(line, sizeof(line), fp)) {
        if (strncmp(line, "MemTotal:", 9) == 0) {
            char *p = line + 9;
            while (*p == ' ') p++;
            total_kb = (uint32_t)strtoul(p, NULL, 10);
            break;
        }
    }
    fclose(fp);
    return total_kb / 1024;
}

uint32_t get_total_ram_mb(void) {
    long pages = sysconf(_SC_PHYS_PAGES);
    long page_size = sysconf(_SC_PAGESIZE);
    if (pages != -1 && page_size != -1 && pages > 0 && page_size > 0) {
        uint64_t total_bytes = (uint64_t)pages * (uint64_t)page_size;
        return (uint32_t)(total_bytes / (1024 * 1024));
    }
    return get_total_ram_mb_fallback();
}

uint8_t get_cpu_cores(void) {
    // Contar directamente desde /proc/cpuinfo (es más fiable que sysconf)
    FILE *fp = fopen("/proc/cpuinfo", "r");
    if (!fp) {
        // Fallback: sysconf
        long cores = sysconf(_SC_NPROCESSORS_CONF);
        if (cores <= 0) cores = 1;
        if (cores > 255) cores = 255;
        return (uint8_t)cores;
    }

    long cores = 0;
    char line[256];
    while (fgets(line, sizeof(line), fp)) {
        if (strncmp(line, "processor", 9) == 0)
            cores++;
    }
    fclose(fp);

    if (cores <= 0) cores = 1;
    if (cores > 255) cores = 255;
    return (uint8_t)cores;
}
