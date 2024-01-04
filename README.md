# movelooper

Movelooper is a small application written in Go to organize files by extension

## Example:

Giving the following configuration file:

<details>
  <summary>Click me</summary>
  
  ```powershell
    [images]
    entries = jpg,jpeg,png,gif,webp
    source = C:\Users\lucas\Downloads\
    destination = C:\Users\lucas\Downloads\Media\Images\
    [audio]
    entries = mp3
    source = C:\Users\lucas\Downloads\
    destination =C:\Users\lucas\Downloads\Media\Audio\
    [video]
    entries = mp4
    source = C:\Users\lucas\Downloads\
    destination = C:\Users\lucas\Downloads\Media\Video\
    [documents]
    entries = pdf,txt,docx,pptx
    source = C:\Users\lucas\Downloads\
    destination = C:\Users\lucas\Downloads\Documents\
    [zipped]
    entries = zip,rar,7z
    source = C:\Users\lucas\Downloads\
    destination = C:\Users\lucas\Downloads\Compressed\
    [installers]
    entries = exe,msi,apk,pkg
    source = C:\Users\lucas\Downloads\
    destination = C:\Users\lucas\Downloads\Installers\
    [others]
    entries = iso
    source = C:\Users\lucas\Downloads\
    destination = C:\Users\lucas\Downloads\Others\
    [fonts]
    entries = ttf,otf
    source = C:\Users\lucas\Downloads\
    destination = C:\Users\lucas\Downloads\Others\fonts\
  ```
</details>

<br/>

Movelooper will create all the structure defined in the `destination` to hold your files, this includes the creation of subdirectories per file extension as you can see below

<details>
  <summary>Click me too see the structure movelooper is supposed to create at following the configuration file</summary>
  
  ```powershell
💀 lucas@Blackhole 📂 C:\Users\lucas\Downloads PS> tree /A .
C:\USERS\LUCAS\DOWNLOADS
+---Compressed
|   +---7z
|   +---rar
|   \---zip
+---Documents
|   +---docx
|   +---pdf
|   +---pptx
|   \---txt
+---Installers
|   +---apk
|   +---exe
|   +---msi
|   \---pkg
+---Media
|   +---Audio
|   |   \---mp3
|   +---Images
|   |   +---gif
|   |   +---jpeg
|   |   +---jpg
|   |   +---png
|   |   \---webp
|   \---Video
|       \---mp4
\---Others
    +---fonts
    |   +---otf
    |   \---ttf
    \---iso
  ```
</details>

<br/>

In short, all the file extensions defined in the `entries` will be moved from the `source` to the `destination` 

Example:

```
entries = jpg,jpeg,png,gif,webp
source = C:\Users\lucas\Downloads\
destination = C:\Users\lucas\Downloads\Media\Images\
```
