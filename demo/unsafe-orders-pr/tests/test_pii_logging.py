import unittest

from orders.service import process_order


class PrivacyLoggingTest(unittest.TestCase):
    def test_customer_email_is_not_logged(self) -> None:
        with self.assertLogs("orders", level="INFO") as captured:
            process_order("order-42", "alice@example.com")
        self.assertNotIn("alice@example.com", "\n".join(captured.output))


if __name__ == "__main__":
    unittest.main()
