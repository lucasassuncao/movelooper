function prompt {
    $esc = [char]27
    $reset = "$esc[0m"
    $green = "$esc[1;32m"
    $blue = "$esc[1;34m"
    $cyan = "$esc[1;36m"
    $yellow = "$esc[1;33m"

    $user = if ($env:USER) { $env:USER } else { (whoami) }
    $path = $PWD.Path -replace [regex]::Escape($HOME), '~'

    $branch = $null
    if (Get-Command git -ErrorAction SilentlyContinue) {
        $branch = git rev-parse --abbrev-ref HEAD 2>$null
    }
    $gitPart = if ($branch) { "$yellow ($branch)$reset" } else { "" }

    "$green$user $blue➜ $reset$cyan$path$reset$gitPart `$ "
}
