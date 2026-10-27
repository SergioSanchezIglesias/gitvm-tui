import unittest

from homebrew import formula


class FormulaTests(unittest.TestCase):
    def setUp(self):
        self.checksums = "\n".join(
            f"{digit * 64}  gitvm_0.1.0_{os}_{arch}.tar.gz"
            for digit, os, arch in [
                ("1", "darwin", "amd64"), ("2", "darwin", "arm64"),
                ("3", "linux", "amd64"), ("4", "linux", "arm64"),
            ]
        )

    def test_formula_covers_all_release_platforms(self):
        result = formula("v0.1.0", self.checksums)
        self.assertIn('version "0.1.0"', result)
        for line in self.checksums.splitlines():
            checksum, filename = line.split()
            self.assertIn(f'/releases/download/v0.1.0/{filename}"', result)
            self.assertIn(f'sha256 "{checksum}"', result)
        self.assertIn('bin.install "gitvm"', result)
        self.assertIn('system "#{bin}/gitvm", "--help"', result)

    def test_missing_or_corrupt_archives_are_rejected(self):
        for checksums in [
            self.checksums.split("\n", 1)[1],
            self.checksums.replace("1" * 64, "bad"),
            self.checksums + "\n" + self.checksums.splitlines()[0],
        ]:
            with self.subTest(checksums=checksums), self.assertRaises(ValueError):
                formula("v0.1.0", checksums)

    def test_only_stable_semver_tags_are_accepted(self):
        for tag in ["main", "v1.0.0-rc1", "v01.0.0", 'v1.0.0"; system "bad']:
            with self.subTest(tag=tag), self.assertRaises(ValueError):
                formula(tag, self.checksums)


if __name__ == "__main__":
    unittest.main()
