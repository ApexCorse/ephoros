import "dart:math";

import "package:client/components/main_table/main_table.dart";
import "package:client/types/record.dart";
import "package:flutter/material.dart";
import "package:flutter_bloc/flutter_bloc.dart";

class MainTableCubit extends Cubit<MainTableState> {
  MainTableCubit({
    this.columnMinWidth = 100,
    this.labelStyle,
    this.valueStyle,
    this.backgroundColors = const (Colors.white, Color(0xFFF5F5F5)),
    List<Record> elements = const [],
    BoxConstraints constraints = const BoxConstraints(
      maxHeight: 100,
      maxWidth: 100,
    ),
  }) : super(MainTableState(rows: [], columns: 1, constraints: constraints)) {
    if (elements.isNotEmpty) {
      setElements(elements);
    }
  }

  final double columnMinWidth;
  final TextStyle? labelStyle;
  final TextStyle? valueStyle;
  final (Color, Color) backgroundColors;

  void setElements(List<Record> elements) {
    debugPrint("(MainTableCubit) constraints: ${state._constraints}");
    debugPrint("(MainTableCubit) elements: ${elements.length}");

    if (state._constraints.maxWidth < columnMinWidth) {
      emit(state.copyWith(rows: [], columns: 1, elements: elements));
      return;
    }

    final columns = state._constraints.maxWidth ~/ columnMinWidth;
    final actualColumns = max(1, min(columns, elements.length));
    debugPrint("(MainTableCubit) columns: $actualColumns");

    final rows = <Widget>[];
    bool inverted = false;
    for (int i = 0; i < elements.length; i += actualColumns) {
      final row = Row(
        children: elements
            .sublist(i, min(i + actualColumns, elements.length))
            .map(
              (e) => Flexible(
                child: MainTableElement(
                  data: (e.sensor, e.value),
                  inverted: inverted,
                  labelStyle: labelStyle,
                  valueStyle: valueStyle,
                  backgroundColors: backgroundColors,
                ),
              ),
            )
            .toList(),
      );

      for (int i = row.children.length; i < actualColumns; i++) {
        row.children.add(const Flexible(child: SizedBox.shrink()));
      }

      rows.add(row);

      inverted = !inverted;
    }

    emit(
      state.copyWith(rows: rows, columns: actualColumns, elements: elements),
    );
  }

  void updateConstraints(BoxConstraints constraints) {
    if (state._constraints == constraints) {
      return;
    }

    emit(state.copyWith(constraints: constraints));

    setElements(state._elements);
  }
}

@immutable
final class MainTableState {
  const MainTableState({
    required this.rows,
    required int columns,
    required BoxConstraints constraints,
    List<Record> elements = const [],
  }) : _elements = elements,
       _constraints = constraints,
       _columns = columns;

  final List<Widget> rows;

  final int _columns;
  final List<Record> _elements;
  final BoxConstraints _constraints;

  MainTableState copyWith({
    List<Widget>? rows,
    int? columns,
    List<Record>? elements,
    BoxConstraints? constraints,
  }) => MainTableState(
    rows: rows ?? this.rows,
    columns: columns ?? _columns,
    elements: elements ?? _elements,
    constraints: constraints ?? _constraints,
  );
}
