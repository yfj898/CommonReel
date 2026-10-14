"""Package tracked and non-ignored source; never include raw data or local state."""
from pathlib import Path
import hashlib
import json
import subprocess
import zipfile

root = Path(__file__).resolve().parents[1]
paths = sorted(set(subprocess.check_output(["git", "ls-files", "--cached", "--others", "--exclude-standard", "-z"], cwd=root).decode("utf-8").split("\0")))
secrets = []
envfile = root / '.env.local'
if envfile.exists():
    for line in envfile.read_text(encoding='utf-8-sig').splitlines():
        if line.strip().startswith('WATCHMODE_API_KEY='):
            value = line.split('=', 1)[1].strip().strip('"\'')
            if value:
                secrets.append(value.encode())
output_dir = root / "dist"
output_dir.mkdir(exist_ok=True)
output = output_dir / "CommonReel-source.zip"
count = 0
with zipfile.ZipFile(output, "w", zipfile.ZIP_DEFLATED) as archive:
    for name in paths:
        if not name:
            continue
        path = (root / name).resolve()
        if not path.is_relative_to(root) or name.split("/")[0] in {"data", "bin", ".runtime", "dist", ".git"}:
            raise RuntimeError(f"Unexpected package path: {name}")
        if path.name in {'.env', '.env.local'} or path.suffix.lower() in {'.db','.exe','.csv','.log','.zip'}:
            raise RuntimeError(f"Private/binary artifact blocked: {name}")
        content = path.read_bytes()
        if any(value in content for value in secrets):
            raise RuntimeError(f"Credential detected in source: {name}")
        archive.write(path, "CommonReel/" + name)
        count += 1
print(json.dumps({"path": str(output), "files": count, "bytes": output.stat().st_size,
                  "sha256": hashlib.sha256(output.read_bytes()).hexdigest()}, indent=2))
