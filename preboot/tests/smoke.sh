#!/bin/sh
set -eu
cd "$(dirname "$0")/.."
sh build.sh --test
disk=$(mktemp -d)
trap 'rm -rf "$disk"' EXIT
mkdir -p "$disk/EFI/BOOT" "$disk/EFI/Microsoft/Boot"
cp build/SilkSplash.efi "$disk/EFI/BOOT/BOOTX64.EFI"
cp build/HandoverTest.efi "$disk/EFI/Microsoft/Boot/bootmgfw.efi"
cp /usr/share/OVMF/OVMF_VARS_4M.fd build/variables.fd
status=0
timeout 60 qemu-system-x86_64 -machine q35,accel=tcg -m 256 -display none -net none -no-reboot \
 -drive if=pflash,format=raw,readonly=on,file=/usr/share/OVMF/OVMF_CODE_4M.fd \
 -drive if=pflash,format=raw,file=build/variables.fd \
 -drive "format=raw,file=fat:rw:$disk" -device isa-debug-exit,iobase=0xf4,iosize=4 || status=$?
test "$status" -eq 43 || { echo "Firmware handover failed (QEMU exit $status)";exit 1; }
