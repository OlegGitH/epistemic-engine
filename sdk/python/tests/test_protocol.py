import json
import sys
import unittest
from pathlib import Path

ROOT = Path(__file__).resolve().parents[3]
sys.path.insert(0, str(ROOT / "sdk" / "python" / "src"))
from epistemic_protocol import hash_value, validate_event

class ProtocolTest(unittest.TestCase):
    def test_canonical_hash(self):
        fixture = json.loads((ROOT / "conformance" / "fixtures" / "canonical.json").read_text())
        self.assertEqual(hash_value(fixture["value"]), fixture["sha256"])

    def test_valid_event(self):
        event = json.loads((ROOT / "conformance" / "fixtures" / "valid-event.json").read_text())
        validate_event(event)

if __name__ == "__main__": unittest.main()
