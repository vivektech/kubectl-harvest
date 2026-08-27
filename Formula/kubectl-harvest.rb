# typed: false
# frozen_string_literal: true

# This file is generated and updated by GoReleaser when a release is
# published. The checksums below are placeholders until the first release
# of this project is cut.
class KubectlHarvest < Formula
  desc "kubectl plugin that deletes unused Kubernetes resources"
  homepage "https://github.com/vivektech/kubectl-harvest"
  version "1.0.0"
  license "Apache-2.0"

  on_macos do
    if Hardware::CPU.intel?
      url "https://github.com/vivektech/kubectl-harvest/releases/download/v1.0.0/kubectl-harvest_1.0.0_darwin_amd64.tar.gz"
      sha256 "0000000000000000000000000000000000000000000000000000000000000000"
    end
    if Hardware::CPU.arm? && Hardware::CPU.is_64_bit?
      url "https://github.com/vivektech/kubectl-harvest/releases/download/v1.0.0/kubectl-harvest_1.0.0_darwin_arm64.tar.gz"
      sha256 "0000000000000000000000000000000000000000000000000000000000000000"
    end
  end
  on_linux do
    if Hardware::CPU.intel?
      url "https://github.com/vivektech/kubectl-harvest/releases/download/v1.0.0/kubectl-harvest_1.0.0_linux_amd64.tar.gz"
      sha256 "0000000000000000000000000000000000000000000000000000000000000000"
    end
    if Hardware::CPU.arm? && Hardware::CPU.is_64_bit?
      url "https://github.com/vivektech/kubectl-harvest/releases/download/v1.0.0/kubectl-harvest_1.0.0_linux_arm64.tar.gz"
      sha256 "0000000000000000000000000000000000000000000000000000000000000000"
    end
  end

  def install
    bin.install "kubectl-harvest"
  end

  test do
    assert_match version.to_s, shell_output("#{bin}/kubectl-harvest --version 2>&1")
  end
end
