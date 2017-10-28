New-Item -ItemType Directory -Path ./build
Set-Location ./build

gox -osarch="windows/amd64" ../

Move-Item whalebrew_windows_amd64.exe whalebrew-Windows-x86_64.exe
