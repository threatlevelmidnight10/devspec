class Devspec < Formula
  desc "Deterministic CLI orchestration for Cursor agent workflows"
  homepage "https://github.com/threatlevelmidnight10/devspec"
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
