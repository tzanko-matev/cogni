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
      mkPkgs = system: import nixpkgs { inherit system; };
      mkCogni = system:
        let
          pkgs = mkPkgs system;
        in
        pkgs.buildGoModule {
          pname = "cogni";
          version = "0.1.0";
          src = ./.;
          subPackages = [ "cmd/cogni" ];
          vendorHash = "sha256-hsk6ryKBE9hbcU8ccsLgCA0jAqJJnA7gYK0KSIA75q8=";
        };
    in
    {
      packages = forAllSystems (system: {
        cogni = mkCogni system;
        default = mkCogni system;
      });
      devShells = forAllSystems (system:
        let
          pkgs = mkPkgs system;
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
              (mkCogni system)
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
