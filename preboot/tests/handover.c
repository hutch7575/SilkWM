// OVMF-only fixture: signal that firmware entered the test image successfully.
#include <efi.h>
EFI_STATUS efi_main(EFI_HANDLE image, EFI_SYSTEM_TABLE *system) {
    (void)image; (void)system;
    __asm__ volatile("outl %0,%1" : : "a"(21), "Nd"((unsigned short)0xf4));
    return EFI_SUCCESS;
}
