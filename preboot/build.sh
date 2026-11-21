#!/bin/sh
set -eu
cd "$(dirname "$0")"
out=${OUTPUT_DIR:-build}
inc=${EFI_INCLUDE:-/usr/include/efi}
lib=${EFI_LIB:-/usr/lib}
mkdir -p "$out"
python3 ribbon.py > "$out/ribbon.h"
build_efi() {
    gcc -Os -Wall -Wextra -Werror -ffreestanding -fpic -fshort-wchar -mno-red-zone -fno-stack-protector -fno-stack-check -DGNU_EFI_USE_MS_ABI -I"$inc" -I"$inc/x86_64" -I"$out" -c "$1" -o "$out/$2.o"
    ld -nostdlib -znocombreloc -T "$lib/elf_x86_64_efi.lds" -shared -Bsymbolic "$lib/crt0-efi-x86_64.o" "$out/$2.o" -L"$lib" -lefi -lgnuefi -o "$out/$2.so"
    objcopy -j .text -j .sdata -j .data -j .rodata -j .dynamic -j .dynsym -j .rel -j .rela -j .reloc --target=efi-app-x86_64 "$out/$2.so" "$out/$2.efi"
}
build_efi splash.c SilkSplash
if [ "${1:-}" = "--test" ]; then build_efi tests/handover.c HandoverTest; fi
