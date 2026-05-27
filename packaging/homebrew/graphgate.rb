class Graphgate < Formula
  desc "CLI-first GraphQL contract gate with a local web UI"
  homepage "https://github.com/MarlonJD/graphgate"
  version "0.1.1"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/MarlonJD/graphgate/releases/download/v#{version}/graphgate_darwin_arm64.tar.gz"
      sha256 "1742e6eb21fc4afbcdc78a863b6e537190a1c9aad658c2c0d49e2669acb580d0"
    else
      url "https://github.com/MarlonJD/graphgate/releases/download/v#{version}/graphgate_darwin_amd64.tar.gz"
      sha256 "db0f318fcb4256672d682356ddb37ea207326553c5f6a5a8ef78492450672bc3"
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "https://github.com/MarlonJD/graphgate/releases/download/v#{version}/graphgate_linux_arm64.tar.gz"
      sha256 "c59be54efdf63a0e24cde31535c3a9599a980b484d9f8badd56c765b92dd0607"
    else
      url "https://github.com/MarlonJD/graphgate/releases/download/v#{version}/graphgate_linux_amd64.tar.gz"
      sha256 "4b025f780720675a392a69ae40a632f2e752775c9bf4ed2147c30d3c25998ab8"
    end
  end

  def install
    bin.install "graphgate"
  end

  test do
    system "#{bin}/graphgate", "help"
  end
end
