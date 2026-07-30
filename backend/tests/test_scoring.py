import unittest
from datetime import date

from sqlalchemy import create_engine
from sqlalchemy.orm import sessionmaker

from database import Base
from models import District, KpiCatalog, KpiEntry, User
from services.scoring import recalculate_officer_month


class MonthlyScoringTests(unittest.TestCase):
    def setUp(self):
        engine = create_engine(
            "sqlite:///:memory:",
            connect_args={"check_same_thread": False},
        )
        Base.metadata.create_all(bind=engine)
        self.db = sessionmaker(bind=engine)()
        district = District(name="Test", coeff=1.0)
        self.officer = User(
            username="officer",
            password_hash="test",
            full_name="Төлегенов Н",
            rank="Инспектор",
            role="officer",
            district=district,
            monthly_norm=100,
        )
        self.db.add_all([district, self.officer])
        self.db.flush()

    def tearDown(self):
        self.db.close()

    def add_catalog(
        self,
        code,
        points,
        monthly_limit,
        *,
        is_mass=False,
        is_critical=False,
    ):
        item = KpiCatalog(
            code=code,
            indicator=code,
            block="Test",
            points_per_unit=points,
            monthly_limit=monthly_limit,
            is_mass=is_mass,
            is_critical=is_critical,
        )
        self.db.add(item)
        self.db.flush()
        return item

    def add_entry(self, catalog, quantity):
        entry = KpiEntry(
            date=date(2026, 7, 15),
            month="2026-07",
            officer_id=self.officer.id,
            catalog_id=catalog.id,
            quantity=quantity,
            points_per_unit=catalog.points_per_unit,
            coeff=1.0,
            status="confirmed",
        )
        self.db.add(entry)
        self.db.flush()
        return entry

    def test_monthly_limits_preserve_reported_quantity(self):
        a590_1 = self.add_catalog(
            "A590-1",
            1,
            20,
            is_mass=True,
        )
        a590_3 = self.add_catalog("A590-3", 5, 6)
        entry_1 = self.add_entry(a590_1, 35)
        entry_2 = self.add_entry(a590_3, 8)

        result = recalculate_officer_month(
            self.db,
            self.officer.id,
            "2026-07",
        )

        self.assertEqual(entry_1.quantity, 35)
        self.assertEqual(entry_1.counted_quantity, 20)
        self.assertEqual(entry_1.capped_score, 20)
        self.assertEqual(entry_2.quantity, 8)
        self.assertEqual(entry_2.counted_quantity, 6)
        self.assertEqual(entry_2.capped_score, 30)
        self.assertEqual(result["final_score"], 50)

    def test_mass_filter_uses_capped_positive_scores(self):
        mass = self.add_catalog("MASS", 1, 30, is_mass=True)
        other = self.add_catalog("OTHER", 5, 6)
        self.add_entry(mass, 30)
        self.add_entry(other, 6)

        result = recalculate_officer_month(
            self.db,
            self.officer.id,
            "2026-07",
        )

        self.assertEqual(result["mass_share"], 0.5)
        self.assertEqual(result["anti_coeff"], 0.85)
        self.assertEqual(result["final_score"], 51)

    def test_critical_indicator_zeroes_month(self):
        ordinary = self.add_catalog("ORDINARY", 10, 10)
        critical = self.add_catalog(
            "CRITICAL",
            -100,
            1,
            is_critical=True,
        )
        first = self.add_entry(ordinary, 5)
        second = self.add_entry(critical, 1)

        result = recalculate_officer_month(
            self.db,
            self.officer.id,
            "2026-07",
        )

        self.assertTrue(result["critical"])
        self.assertEqual(result["final_score"], 0)
        self.assertEqual(first.final_score, 0)
        self.assertEqual(second.final_score, 0)


if __name__ == "__main__":
    unittest.main()
