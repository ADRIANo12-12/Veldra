SHELL := /bin/bash
ROOT  := $(CURDIR)

VERSION := $(shell scripts/version.sh show)
CHANNEL := $(shell scripts/version.sh channel)
STATUS  := $(shell scripts/version.sh status)
ARCH    := $(shell scripts/version.sh arch)
ISO     := build/out/veldra-$(VERSION)-$(ARCH).iso

.PHONY: all help plan tui rootfs iso qemu container replit-iso replit-qemu \
        test check check-deps lint fmt keyring \
        clean version

all: help

help: ## Show available targets
	@printf 'Veldra $(VERSION) — development targets\n\n'
	@grep -E '^[a-zA-Z_-]+:.*##' $(MAKEFILE_LIST) \
		| awk -F ':.*## ' '{printf "  \033[1;36m%-14s\033[0m %s\n", $$1, $$2}'

# --- build ----------------------------------------------------------------
plan: ## Print the end-to-end build plan (builds nothing)
	build/build.sh plan

tui: ## Build the TUI binary (static, version-injected)
	scripts/build-tui.sh

rootfs: ## Build the Arch rootfs + live initramfs (in the Arch container)
	build/build.sh rootfs

iso: rootfs ## Build the bootable live ISO (in the Arch container)
	build/build.sh iso

all: ## tui + rootfs + iso
	build/build.sh all

container: ## Build/refresh the isolated Arch build container image
	build/container/run.sh --init

replit-iso: tui ## Build a Replit-compatible ISO without Docker/Podman/root
	@command -v nix >/dev/null 2>&1 || { echo "nix is required for replit-iso"; exit 1; }
	@nix shell nixpkgs#xorriso nixpkgs#squashfsTools --command bash -c 'cd "$(ROOT)" && bash build/replit-iso.sh'

replit-qemu: replit-iso ## Boot the Replit-compatible ISO headless using QEMU TCG
	@test -f "$(ISO)" || { echo "ISO not found: $(ISO)"; exit 1; }
	@echo "Booting Veldra $(VERSION) in QEMU/TCG (headless)..."
	@qemu-system-x86_64 \
		-machine accel=tcg \
		-m 1024 \
		-nographic \
		-display none \
		-serial mon:stdio \
		$(VELDRA_QEMU_ARGS) \
		-cdrom "$(ISO)"

qemu: ## Boot the built ISO headless (QEMU, -nographic)
	@test -f "$(ISO)" || { echo "ISO not found: $(ISO) — run 'make iso' first"; exit 1; }
	@echo "Booting Veldra $(VERSION) in QEMU (headless)..."
	@qemu-system-x86_64 \
		-m 1024 \
		-nographic \
		$(VELDRA_QEMU_ARGS) \
		-cdrom "$(ISO)"

# --- verification ----------------------------------------------------------
test: ## Run the Go unit tests
	cd $(ROOT)/tui && go test ./...

check: ## Full tree check: compile, vet, tests, shell syntax, template sanity
	scripts/build-tui.sh --check
	cd $(ROOT)/tui && go test ./...
	bash -n scripts/common.sh scripts/version.sh scripts/check-deps.sh scripts/build-tui.sh
	bash -n build/build.sh build/rootfs.sh build/iso.sh build/container/run.sh build/replit-iso.sh
	bash -n system/install.sh boot/install.sh
	scripts/version.sh inject boot/grub/grub.cfg.in >/dev/null
	@echo "[ OK ] Veldra $(VERSION) checks passed"

check-deps: ## Check host prerequisites (Go, Podman/Docker, QEMU, git, make)
	scripts/check-deps.sh

lint: ## Lint the shell scripts (shellcheck)
	@command -v shellcheck >/dev/null 2>&1 || { echo "shellcheck is not installed"; exit 1; }
	shellcheck -S warning scripts/*.sh build/*.sh build/container/*.sh system/*.sh boot/*.sh
	shellcheck -S warning build/replit-iso.sh

fmt: ## Format the Go source
	cd $(ROOT)/tui && gofmt -l . | grep -v '^$$' || true
	cd $(ROOT)/tui && gofmt -w .

# --- clean ----------------------------------------------------------------
clean: ## Remove build artifacts and work products
	build/build.sh clean

version: ## Print the current Veldra version and identity
	scripts/version.sh identity
