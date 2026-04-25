def obfuscate(s, key):
    k1 = key & 0xff
    k2 = (key >> 8) & 0xff
    k3 = (key >> 16) & 0xff
    k4 = (key >> 24) & 0xff
    res = []
    for ch in s:
        b = ord(ch)
        b ^= k1
        b ^= k2
        b ^= k3
        b ^= k4
        res.append(f"\\x{b:02X}")
    return ''.join(res)



domain = "cnc-server.xyz\x00"  # Reemplaza con tu dominio
print("Obfuscated domain:")
print(obfuscate(domain, 0xdeadbeef))


ip = "185.242.3.160\x00"  # Reemplaza con tu IP
print("Obfuscated IP:")
print(obfuscate(ip, 0xdeadbeef))

