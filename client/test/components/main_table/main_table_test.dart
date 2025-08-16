import "package:client/components/main_table/main_table.dart";
import "package:client/components/main_table/main_table_cubit.dart";
import "package:client/types/record.dart";
import "package:flutter/material.dart";
import "package:flutter_test/flutter_test.dart";

void main() {
  group("MainTable", () {
    testWidgets("should render a table with the correct number of rows", (
      tester,
    ) async {
      final cubit = MainTableCubit(columnMinWidth: 100);
      await tester.pumpWidget(
        MaterialApp(
          home: Scaffold(body: MainTable(cubit: cubit)),
        ),
      );
      await tester.pumpAndSettle();

      expect(find.byType(MainTable), findsOne);
      expect(find.byType(MainTableElement), findsNothing);

      cubit.setElements([
        Record(
          id: 1,
          sensor: "Temperature",
          value: 22.0,
          module: "ModuleA",
          section: "Section1",
          date: DateTime(2023, 1, 1),
        ),
        Record(
          id: 2,
          sensor: "Humidity",
          value: 45.0,
          module: "ModuleB",
          section: "Section2",
          date: DateTime(2023, 1, 2),
        ),
      ]);
      await tester.pumpAndSettle();

      expect(find.byType(MainTableElement), findsNWidgets(2));
      expect(find.text("Temperature"), findsOne);
      expect(find.text("Humidity"), findsOne);
      expect(find.text("ModuleA"), findsNothing);
      expect(find.text("ModuleB"), findsNothing);
      expect(find.text("Section1"), findsNothing);
      expect(find.text("Section2"), findsNothing);
      expect(find.text("2023-01-01"), findsNothing);
      expect(find.text("2023-01-02"), findsNothing);
      expect(find.text("22.0"), findsOne);
      expect(find.text("45.0"), findsOne);
    });
  });
}
