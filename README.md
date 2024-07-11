# movelooper

Movelooper is a small application written in Go to organize files by extension

In essence, files with extensions listed in `entries` will be moved from `source` to `destination`, each within a corresponding folder named after its extension.

```
entries = jpg,jpeg,png,gif,webp
source = C:\Users\lucas\Downloads\
destination = C:\Users\lucas\Downloads\Media\Images\
```

## Example:

Giving the following configuration file:

<details>
  <summary>Click to expand</summary>

  ```powershell
    [documents]
    entries = pdf,txt,docx,pptx
    source = C:\Users\lucas\Downloads\
    destination = C:\Users\lucas\Downloads\Documents\
  ```
</details>

Movelooper will create all the structure defined in the `destination` to hold your files, this includes the creation of subdirectories per file extension as you can see below

<details>
  <summary>Click to expand</summary>
  
```powershell
💀 lucas@Blackhole 📂 C:\Users\lucas\Downloads PS> tree /A .
C:\USERS\LUCAS\DOWNLOADS
+---Documents
|   +---docx
|   +---pdf
|   +---pptx
|   \---txt
```
</details>


## Output Example:

<details>
  <summary>Click to expand</summary>
  
```powershell
💀 lucas@Blackhole 📂 C:\Users\lucas PS> movelooper.exe
2024/07/10 22:01:10 [INFO] Nenhum arquivo .jpg para mover
2024/07/10 22:01:10 [INFO] Nenhum arquivo .jpeg para mover
2024/07/10 22:01:10 [INFO] Nenhum arquivo .png para mover
2024/07/10 22:01:10 [INFO] Nenhum arquivo .gif para mover
2024/07/10 22:01:10 [INFO] Nenhum arquivo .webp para mover
2024/07/10 22:01:10 [INFO] Nenhum arquivo .mp3 para mover
2024/07/10 22:01:10 [INFO] Nenhum arquivo .mp4 para mover
2024/07/10 22:01:10 [INFO] Nenhum arquivo .pdf para mover
2024/07/10 22:01:10 [INFO] Nenhum arquivo .txt para mover
2024/07/10 22:01:10 [INFO] Nenhum arquivo .docx para mover
2024/07/10 22:01:10 [INFO] Nenhum arquivo .pptx para mover
2024/07/10 22:01:10 [INFO] Nenhum arquivo .zip para mover
2024/07/10 22:01:10 [INFO] Nenhum arquivo .rar para mover
2024/07/10 22:01:10 [INFO] 3 arquivos .7z para mover
2024/07/10 22:01:10 [INFO] Sucesso ao mover o arquivo: C:\Users\lucas\Downloads\FFMQ_PT-BR_V1_0.7z.
2024/07/10 22:01:10 [INFO] Sucesso ao mover o arquivo: C:\Users\lucas\Downloads\NSudo_Launcher_Installer_AIO.7z.
2024/07/10 22:01:10 [INFO] Sucesso ao mover o arquivo: C:\Users\lucas\Downloads\geek.7z.
2024/07/10 22:01:10 [INFO] Nenhum arquivo .exe para mover
2024/07/10 22:01:10 [INFO] Nenhum arquivo .msi para mover
2024/07/10 22:01:10 [INFO] 7 arquivos .apk para mover
2024/07/10 22:01:10 [INFO] Sucesso ao mover o arquivo: C:\Users\lucas\Downloads\AdAway-6.1.3-20240706.apk.
2024/07/10 22:01:10 [INFO] Sucesso ao mover o arquivo: C:\Users\lucas\Downloads\JoiPlay-1.20.027-a320fc05.apk.
2024/07/10 22:01:10 [INFO] Sucesso ao mover o arquivo: C:\Users\lucas\Downloads\RPGMPlugin-1.20.28-2d33a054.apk.
2024/07/10 22:01:10 [INFO] Sucesso ao mover o arquivo: C:\Users\lucas\Downloads\halo-1-4.apk.
2024/07/10 22:01:10 [INFO] Sucesso ao mover o arquivo: C:\Users\lucas\Downloads\malody-4-3-7.apk.
2024/07/10 22:01:10 [INFO] Sucesso ao mover o arquivo: C:\Users\lucas\Downloads\melobeat-1-7-10.apk.
2024/07/10 22:01:10 [INFO] Sucesso ao mover o arquivo: C:\Users\lucas\Downloads\rom-toolbox-lite-6-5-3-0.apk.
2024/07/10 22:01:10 [INFO] Nenhum arquivo .pkg para mover
2024/07/10 22:01:10 [INFO] Nenhum arquivo .iso para mover
2024/07/10 22:01:10 [INFO] Nenhum arquivo .ttf para mover
2024/07/10 22:01:10 [INFO] Nenhum arquivo .otf para mover
```
</details>