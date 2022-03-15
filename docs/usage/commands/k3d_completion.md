## k3d completion

Generate completion scripts for [bash, zsh, fish, powershell | psh]

### Synopsis

To load completions:

Bash:

 $ source <(k3d completion bash)

 # To load completions for each session, execute once:
 # Linux:
 $ k3d completion bash > /etc/bash_completion.d/k3d
 # macOS:
 $ k3d completion bash > /usr/local/etc/bash_completion.d/k3d

Zsh:

 # If shell completion is not already enabled in your environment,
 # you will need to enable it.  You can execute the following once:

 $ echo "autoload -U compinit; compinit" >> ~/.zshrc

 # To load completions for each session, execute once:
 $ k3d completion zsh > "${fpath[1]}/_k3d"

 # You will need to start a new shell for this setup to take effect.

fish:

 $ k3d completion fish | source

 # To load completions for each session, execute once:
 $ k3d completion fish > ~/.config/fish/completions/k3d.fish

PowerShell:

 PS> k3d completion powershell | Out-String | Invoke-Expression

 # To load completions for every new session, run:
 PS> k3d completion powershell > k3d.ps1
 # and source this file from your PowerShell profile.

```
k3d completion SHELL
```

### Options

```
  -h, --help   help for completion
```

### Options inherited from parent commands

```
      --timestamps   Enable Log timestamps
      --trace        Enable super verbose output (trace logging)
      --verbose      Enable verbose output (debug logging)
```

### SEE ALSO

* [k3d](k3d.md)  - <https://k3d.io/> -> Run k3s in Docker!
