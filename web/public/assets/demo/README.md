# Guided demo narration

These MP3 files are prerecorded release assets. Navy mode uses the local custom
Jarvis Pocket TTS voice; Pirate mode uses the local custom Captain Barbossa
voice. The browser does not contact TTS while a guided demo is running.

The canonical narration text and sequence live in `web/src/guidedDemo.ts`.
Regenerate intentionally from the repository root with:

```powershell
python scripts/generate_demo_narration.py --base-url http://192.168.50.214:8080
```

Render only intentionally changed beats by repeating `--beat`, for example:

```powershell
python scripts/generate_demo_narration.py --beat 06-execution --beat 08-resilience
```

Limit regeneration to one persona when only one voice has a defect:

```powershell
python scripts/generate_demo_narration.py --beat 06-execution --persona navy
```

Long narration is synthesized in sentence-bounded chunks before being joined into
one MP3. This avoids phrase-looping artifacts from cloned voices while preserving
natural cadence.

The voice models remain private runtime assets on VM 214 and are not committed.
