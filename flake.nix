{
  description = "cogni dev environment";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixpkgs-unstable";
  };

  outputs = { self, nixpkgs }:
    let
      systems = [
        "x86_64-linux"
        "aarch64-linux"
        "x86_64-darwin"
        "aarch64-darwin"
      ];
      forAllSystems = f: nixpkgs.lib.genAttrs systems (system: f system);
    in
    {
      devShells = forAllSystems (system:
        let
          pkgs = import nixpkgs { inherit system; };
        in
        {
          default = pkgs.mkShell {
            packages = with pkgs; [
              go
              gopls
              gotools
              golangci-lint
              git
              jujutsu
              ripgrep
              jq
              bashInteractive
              hugo
            ];
            shellHook = ''
              if [ -z "$LLM_PROVIDER" ]; then
                export LLM_PROVIDER=openrouter
              fi
              if [ -z "$LLM_MODEL" ]; then
                export LLM_MODEL="gpt-4.1-mini"
              fi
              if [ -z "$LLM_API_KEY" ] && [ -n "$OPENROUTER_API_KEY" ]; then
                export LLM_API_KEY="$OPENROUTER_API_KEY"
              fi
              echo "cogni dev shell: set LLM_API_KEY to run benchmarks."
            '';
          };
        });
    };
}
