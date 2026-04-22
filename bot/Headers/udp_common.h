#ifndef UDP_COMMON_H
#define UDP_COMMON_H

void init_rand(unsigned long int x);
unsigned long int rand_cmwc(void);
int randnum(int min_num, int max_num);
char random_hex(void);
unsigned short csum(unsigned short *buf, int count);
extern char *hexPayload;

#endif
