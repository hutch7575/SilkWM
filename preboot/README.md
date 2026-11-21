# Silk preboot splash (optional, x64 UEFI)

SilkSplash.efi is a separate firmware application. It draws a blue silk fold and a moving ribbon highlight, then asks firmware to start the existing Windows Boot Manager on the **same EFI System Partition**. Escape skips the animation. Windows can still display its own spinner, BitLocker, updates and recovery screens afterwards.

The download is unsigned. A machine enforcing Secure Boot needs a build signed by a key its firmware already trusts. Keep Secure Boot enabled and the original Windows Boot Manager entry available. Changing the boot path can prompt BitLocker recovery, so have your recovery key available before trying it on a physical machine.

For an authorised test machine with a trusted build, place the file alongside the existing boot files at **EFI/Silk/SilkSplash.efi** on the Windows EFI System Partition. Choose that file once using the firmware's “Boot from file” option, if offered. The ordinary Windows Boot Manager entry still boots Windows directly. On hardware that supports custom entries, a separate firmware entry can point to the same file. Select the original entry to undo the change. The desktop executable and setup scripts do not install firmware files or edit the boot order.

On Ubuntu x64, install `gnu-efi gcc binutils python3` and run `sh preboot/build.sh`. The result is `preboot/build/SilkSplash.efi`. For the isolated smoke test, install `qemu-system-x86 ovmf` and run `sh preboot/tests/smoke.sh`. It uses a disposable firmware variable store, a temporary FAT volume and a test target that signals a successful handover. A real Windows install and physical firmware require separate testing.
