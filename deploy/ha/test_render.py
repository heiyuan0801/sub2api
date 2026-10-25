import copy
import importlib.util
import pathlib
import unittest

spec = importlib.util.spec_from_file_location(
    "ha_render", pathlib.Path(__file__).with_name("render.py")
)
module = importlib.util.module_from_spec(spec)
spec.loader.exec_module(module)


class InventoryTests(unittest.TestCase):
    def setUp(self):
        import json

        self.inv = json.loads(
            pathlib.Path(__file__).with_name("inventory.example.json").read_text()
        )

    def test_three_independent_domains(self):
        self.assertEqual(len(module.validate(self.inv)), 3)
        for change in ("duplicate_ip", "no_witness", "public_ip", "bad_id", "db_budget", "redis_budget"):
            inv = copy.deepcopy(self.inv)
            if change == "duplicate_ip":
                inv["nodes"][2]["private_ip"] = inv["nodes"][0]["private_ip"]
            if change == "no_witness":
                inv["nodes"][2]["kind"] = "application"
            if change == "public_ip":
                inv["nodes"][1]["private_ip"] = "8.8.8.8"
            if change == "bad_id":
                inv["nodes"][1]["id"] = "bad\nnode"
            if change == "db_budget":
                inv["capacity"]["database_per_node"] = 200
            if change == "redis_budget":
                inv["capacity"]["redis_pool_per_node"] = 4000
            with self.subTest(change=change), self.assertRaises(ValueError):
                module.validate(inv)


if __name__ == "__main__":
    unittest.main()
