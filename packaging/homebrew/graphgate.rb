class Graphgate < Formula
  desc "CLI-first GraphQL contract gate with a local web UI"
  homepage "https://github.com/MarlonJD/graphgate"
  version "0.1.0"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/MarlonJD/graphgate/releases/download/v#{version}/graphgate_darwin_arm64.tar.gz"
      sha256 "23130d8d77a4a958c3f90c9c83439f8f385798e7af4c6c9382ea1682c813d1bf"
    else
      url "https://github.com/MarlonJD/graphgate/releases/download/v#{version}/graphgate_darwin_amd64.tar.gz"
      sha256 "865544b647b15c304891606235df203818d378b62286b5bd503a0ebe3a4e5b45"
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "https://github.com/MarlonJD/graphgate/releases/download/v#{version}/graphgate_linux_arm64.tar.gz"
      sha256 "9cc4d52039fa0e3ac09e0b3cfe3133c4de791da9f28a9c7a3cd16a9fc9bbd881"
    else
      url "https://github.com/MarlonJD/graphgate/releases/download/v#{version}/graphgate_linux_amd64.tar.gz"
      sha256 "67c7266161511eb2018d10f053f7bedcd5551aa302b79b4ddec7c654c31c3661"
    end
  end

  def install
    bin.install "graphgate"
  end

  test do
    system "#{bin}/graphgate", "help"
  end
end
