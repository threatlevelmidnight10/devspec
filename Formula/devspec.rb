class Devspec < Formula
  desc "Deterministic CLI orchestration for Cursor agent workflows"
  homepage "https://github.com/threatlevelmidnight10/devspec"
  url "https://github.com/threatlevelmidnight10/devspec/archive/refs/tags/v0.1.1.tar.gz"
  sha256 "8f2c2b763c0926ba1a78b35b091a08f3aa986531429a46f0c149cc998d86ee08"
  head "https://github.com/threatlevelmidnight10/devspec.git", branch: "main"

  depends_on "go" => :build

  def install
    system "go", "build", *std_go_args(ldflags: "-s -w"), "./cmd/devspec"
  end

  test do
    output = shell_output("#{bin}/devspec 2>&1", 1)
    assert_match "devspec - deterministic agent workflow runner", output
  end
end
