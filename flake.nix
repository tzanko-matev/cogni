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
          python = pkgs.python311;
          pythonEnv = python.withPackages (ps: [
            ps.openai
            ps.pydantic
            ps.prompt-toolkit
            ps.pytest
            ps.rich
          ]);
          pythonSitePackages = "${pythonEnv}/${python.sitePackages}";
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
              pythonEnv
            ];
            shellHook = ''
              # Ensure Nix Python + packages are found even if PATH is reordered by shell init.
              export PATH="${pythonEnv}/bin:$PATH"
              export NIX_PYTHONPATH="${pythonSitePackages}:$NIX_PYTHONPATH"
              if [ -z "$LLM_PROVIDER" ]; then
                export LLM_PROVIDER=openrouter
              fi
              if [ -z "$LLM_MODEL" ]; then
                export LLM_MODEL="gpt-4.1-mini"
              fi
              if [ -z "$LLM_API_KEY" ] && [ -n "$OPENROUTER_API_KEY" ]; then
                export LLM_API_KEY="$OPENROUTER_API_KEY"
              fi
              export PATH="$PWD:$PATH"
              echo "cogni dev shell: set LLM_API_KEY to run benchmarks."
            '';
          };
        });
    };
}
