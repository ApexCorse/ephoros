import "package:client/components/main_table/main_table_cubit.dart";
import "package:flutter/material.dart";
import "package:flutter_bloc/flutter_bloc.dart";

class MainTable extends StatelessWidget {
  const MainTable({required this.cubit, super.key});

  final MainTableCubit cubit;

  @override
  Widget build(BuildContext context) => BlocProvider.value(
    value: cubit,
    child: BlocBuilder<MainTableCubit, MainTableState>(
      builder: (context, state) {
        debugPrint("(MainTable) state.rows: ${state.rows.length}");
        return MainTableView(
          rows: state.rows,
          onConstraintsChanged: context
              .read<MainTableCubit>()
              .updateConstraints,
        );
      },
    ),
  );
}

class MainTableView extends StatelessWidget {
  const MainTableView({
    required this.rows,
    required this.onConstraintsChanged,
    super.key,
  });

  final List<Widget> rows;
  final void Function(BoxConstraints) onConstraintsChanged;

  @override
  Widget build(BuildContext context) => LayoutBuilder(
    builder: (context, constraints) {
      onConstraintsChanged(constraints);

      final children = <Widget>[];
      for (int i = 0; i < rows.length; i++) {
        children.add(rows[i]);
        if (i < rows.length - 1) {
          children.add(
            Container(
              height: 1,
              width: double.infinity,
              color: Colors.grey.shade300,
            ),
          );
        }
      }

      return Container(
        clipBehavior: Clip.hardEdge,
        decoration: BoxDecoration(
          borderRadius: BorderRadius.circular(16),
          //TODO(lentscode): Fix the border in the corners
          border: Border.all(color: Colors.grey.shade300),
        ),
        child: Column(mainAxisSize: MainAxisSize.min, children: children),
      );
    },
  );
}

class MainTableElement extends StatelessWidget {
  const MainTableElement({
    required this.data,
    required this.inverted,
    required this.backgroundColors,
    this.labelStyle,
    this.valueStyle,
    super.key,
  });

  final (String, num) data;
  final bool inverted;
  final TextStyle? labelStyle;
  final TextStyle? valueStyle;
  final (Color, Color) backgroundColors;

  @override
  Widget build(BuildContext context) => DecoratedBox(
    decoration: BoxDecoration(
      color: inverted ? backgroundColors.$1 : backgroundColors.$2,
    ),
    child: Padding(
      padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 8),
      child: Row(
        mainAxisAlignment: MainAxisAlignment.spaceBetween,
        children: [
          Text(data.$1, style: labelStyle),
          Text(data.$2.toString(), style: valueStyle),
        ],
      ),
    ),
  );
}
