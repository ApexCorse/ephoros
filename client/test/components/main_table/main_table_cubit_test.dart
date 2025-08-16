import "package:bloc_test/bloc_test.dart";
import "package:client/components/main_table/main_table_cubit.dart";
import "package:client/types/record.dart";
import "package:flutter/material.dart";
import "package:flutter_test/flutter_test.dart";

void main() {
  group("MainTableCubit", () {
    test("initial state is correct", () {
      final cubit = MainTableCubit(
        elements: [
          Record(
            sensor: "Sensor 1",
            id: 1,
            value: 100,
            module: "Module 1",
            section: "Section 1",
            date: DateTime.now(),
          ),
        ],
      );
      expect(cubit.state.rows.length, 1);
      expect(cubit.state.rows[0], isA<Row>());

      final row = cubit.state.rows[0] as Row;
      expect(row.children.length, 1);
    });

    blocTest(
      "setElements emits a new state with updated children",
      build: () => MainTableCubit(
        columnMinWidth: 100,
        constraints: const BoxConstraints(maxHeight: 100, maxWidth: 100),
      ),
      act: (cubit) => cubit.setElements([
        Record(
          sensor: "Sensor 2",
          id: 2,
          value: 200,
          module: "Module 2",
          section: "Section 2",
          date: DateTime.now(),
        ),
      ]),
      expect: () => [
        isA<MainTableState>()
            .having((state) => state.rows.length, "rows length", 1)
            .having((state) => state.rows[0], "rows[0]", isA<Row>())
            .having(
              (state) => (state.rows[0] as Row).children.length,
              "rows[0].children length",
              1,
            ),
      ],
    );

    blocTest(
      "updateConstraints should change the number of row",
      build: () => MainTableCubit(
        columnMinWidth: 100,
        constraints: const BoxConstraints(maxHeight: 100, maxWidth: 300),
        elements: [
          Record(
            sensor: "Sensor 1",
            id: 1,
            value: 100,
            module: "Module 1",
            section: "Section 1",
            date: DateTime(2024, 6, 1, 12, 0, 0),
          ),
          Record(
            sensor: "Sensor 2",
            id: 2,
            value: 200,
            module: "Module 2",
            section: "Section 2",
            date: DateTime(2024, 6, 1, 13, 0, 0),
          ),
        ],
      ),
      act: (cubit) {
        expect(cubit.state.rows.length, 1);
        cubit.updateConstraints(
          const BoxConstraints(maxHeight: 100, maxWidth: 100),
        );
      },
      expect: () => [
        isA<MainTableState>().having(
          (state) => state.rows.length,
          "rows length",
          1,
        ),
        isA<MainTableState>()
            .having((state) => state.rows.length, "rows length", 2)
            .having((state) => state.rows[0], "rows[0]", isA<Row>())
            .having(
              (state) => (state.rows[0] as Row).children.length,
              "rows[0].children length",
              1,
            )
            .having((state) => state.rows[1], "rows[1]", isA<Row>())
            .having(
              (state) => (state.rows[1] as Row).children.length,
              "rows[1].children length",
              1,
            ),
      ],
    );

    test("emits state without rows if constraints are too strict", () {
      final cubit = MainTableCubit(
        columnMinWidth: 100,
        constraints: const BoxConstraints(maxHeight: 100, maxWidth: 50),
        elements: [
          Record(
            sensor: "Sensor 1",
            id: 1,
            value: 100,
            module: "Module 1",
            section: "Section 1",
            date: DateTime.now(),
          ),
          Record(
            sensor: "Sensor 2",
            id: 2,
            value: 200,
            module: "Module 2",
            section: "Section 2",
            date: DateTime.now(),
          ),
        ],
      );

      expect(cubit.state.rows, isEmpty);
    });

    blocTest(
      "updateConstraints should not emit a new state if constraints are too strict",
      build: () => MainTableCubit(
        columnMinWidth: 100,
        constraints: const BoxConstraints(maxHeight: 100, maxWidth: 300),
        elements: [
          Record(
            sensor: "Sensor 1",
            id: 1,
            value: 100,
            module: "Module 1",
            section: "Section 1",
            date: DateTime.now(),
          ),
          Record(
            sensor: "Sensor 2",
            id: 2,
            value: 200,
            module: "Module 2",
            section: "Section 2",
            date: DateTime.now(),
          ),
        ],
      ),
      act: (cubit) {
        expect(cubit.state.rows.length, 1);
        cubit.updateConstraints(
          const BoxConstraints(maxHeight: 100, maxWidth: 50),
        );
      },
      expect: () => [
        isA<MainTableState>().having(
          (state) => state.rows.length,
          "rows length",
          1,
        ),
        isA<MainTableState>().having(
          (state) => state.rows.length,
          "rows length",
          0,
        ),
      ],
    );
  });
}
