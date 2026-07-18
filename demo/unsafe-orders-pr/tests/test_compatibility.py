import unittest

from orders.migration import OrderStatus, migrate_status


class CompatibilityTest(unittest.TestCase):
    def test_legacy_processing_status_is_preserved(self) -> None:
        self.assertEqual(migrate_status("processing"), OrderStatus.in_progress)


if __name__ == "__main__":
    unittest.main()
