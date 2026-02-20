class Devspec < Formula
  desc "Deterministic CLI orchestration for Cursor agent workflows"
  homepage "https://github.com/threatlevelmidnight10/devspec"
  url "https://github.com/threatlevelmidnight10/devspec/archive/refs/tags/v0.1.0.tar.gz"
  sha256 "68dc5d8fbc6512c9dd7e6a265a7e0899f7dc98d9f4f04894603d521c6cfd21a5"
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
