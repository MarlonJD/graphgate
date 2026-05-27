class Graphgate < Formula
  desc "CLI-first GraphQL contract gate with a local web UI"
  homepage "https://github.com/MarlonJD/graphgate"
  version "0.1.0"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/MarlonJD/graphgate/releases/download/v#{version}/graphgate_darwin_arm64.tar.gz"
      sha256 "replace-with-release-sha256"
    else
      url "https://github.com/MarlonJD/graphgate/releases/download/v#{version}/graphgate_darwin_amd64.tar.gz"
      sha256 "replace-with-release-sha256"
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "https://github.com/MarlonJD/graphgate/releases/download/v#{version}/graphgate_linux_arm64.tar.gz"
      sha256 "replace-with-release-sha256"
    else
      url "https://github.com/MarlonJD/graphgate/releases/download/v#{version}/graphgate_linux_amd64.tar.gz"
      sha256 "replace-with-release-sha256"
    end
  end

  def install
    bin.install "graphgate"
  end

  test do
    system "#{bin}/graphgate", "help"
  end
end
