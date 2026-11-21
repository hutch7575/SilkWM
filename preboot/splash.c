#include <efi.h>
#include <efilib.h>
#include "ribbon.h"

typedef EFI_GRAPHICS_OUTPUT_BLT_PIXEL pixel;

static UINTN distance(INTN a, INTN b) {return a>b ? (UINTN)(a-b) : (UINTN)(b-a);}
static pixel shade(UINTN x, UINTN y, UINTN width, UINTN height) {
    // A diagonal silk fold, with a narrow highlight over a broad blue shadow.
    INTN u=(INTN)(x*512/width)-256;
    INTN ridge=198-u/6-(u*u)/2600;
    UINTN d=distance((INTN)(y*256/height),ridge);
    UINTN glow=1800/(25+d*d/4)+90/(2+d);
    pixel p={(UINT8)(19+glow),(UINT8)(8+glow/3),(UINT8)(5+glow/5),0};
    return p;
}
static void animate(void) {
    EFI_GRAPHICS_OUTPUT_PROTOCOL *gop=NULL;
    if(EFI_ERROR(BS->LocateProtocol(&GraphicsOutputProtocol,NULL,(VOID**)&gop)) || !gop->Mode || !gop->Mode->Info) return;
    UINTN w=gop->Mode->Info->HorizontalResolution,h=gop->Mode->Info->VerticalResolution;
    if(w<160 || h<160 || w>16384 || h>16384) return;
    pixel *band=AllocatePool(w*24*sizeof(pixel));
    if(!band) return;
    for(UINTN top=0;top<h;top+=24) {
        UINTN rows=h-top<24?h-top:24;
        for(UINTN y=0;y<rows;y++) for(UINTN x=0;x<w;x++) band[y*w+x]=shade(x,top+y,w,h);
        if(EFI_ERROR(gop->Blt(gop,band,EfiBltBufferToVideo,0,0,0,top,w,rows,0))) {FreePool(band);return;}
    }
    FreePool(band);
    UINTN size=h/6; if(size>220) size=220;
    UINTN ox=(w-size)/2,oy=(h-size)/2;
    pixel *tile=AllocatePool(size*size*sizeof(pixel));
    if(!tile) return;
    for(UINTN frame=0;frame<38;frame++) {
        EFI_INPUT_KEY key;
        if(ST->ConIn && ST->ConIn->ReadKeyStroke(ST->ConIn,&key)==EFI_SUCCESS && key.ScanCode==SCAN_ESC) break;
        for(UINTN y=0;y<size;y++) for(UINTN x=0;x<size;x++) {
            pixel p=shade(ox+x,oy+y,w,h);
            UINTN mask=ribbon_mask[y*100/size][x*100/size];
            UINTN exposure=frame<14?frame*255/14:255;
            UINTN shine=2200/(12+distance((INTN)(x*90/size+y*30/size),(INTN)frame*4-14));
            UINTN alpha=exposure*mask/255;
            p.Red=(UINT8)((p.Red*(255-alpha)+(85+shine*3/4)*alpha)/255);
            p.Green=(UINT8)((p.Green*(255-alpha)+(138+shine/2)*alpha)/255);
            p.Blue=(UINT8)((p.Blue*(255-alpha)+250*alpha)/255);
            tile[y*size+x]=p;
        }
        if(EFI_ERROR(gop->Blt(gop,tile,EfiBltBufferToVideo,0,0,ox,oy,size,size,0))) break;
        BS->Stall(26000);
    }
    FreePool(tile);
}

EFI_STATUS efi_main(EFI_HANDLE self, EFI_SYSTEM_TABLE *system) {
    InitializeLib(self,system);
    EFI_LOADED_IMAGE *image=NULL;
    EFI_STATUS status=BS->HandleProtocol(self,&LoadedImageProtocol,(VOID**)&image);
    if(EFI_ERROR(status)) return status;
    EFI_DEVICE_PATH *path=FileDevicePath(image->DeviceHandle,L"\\EFI\\Microsoft\\Boot\\bootmgfw.efi");
    if(!path) return EFI_OUT_OF_RESOURCES;
    // Load by device path so firmware performs its usual image authentication.
    EFI_HANDLE windows=NULL;
    status=BS->LoadImage(FALSE,self,path,NULL,0,&windows);
    FreePool(path);
    if(EFI_ERROR(status)) {
        if(windows) BS->UnloadImage(windows);
        Print(L"Silk: Windows Boot Manager unavailable (%r). Use the original Windows boot entry.\r\n",status);
        return status;
    }
    animate();
    return BS->StartImage(windows,NULL,NULL);
}
