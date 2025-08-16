import "package:flutter/foundation.dart";

@immutable
class Record {
  const Record({
    required this.id,
    required this.sensor,
    required this.module,
    required this.section,
    required this.date,
    required this.value,
  });

  factory Record.fromJson(Map<String, dynamic> json) {
    try {
      return Record(
        id:
            int.tryParse(json["id"]?.toString() ?? "") ??
            (throw const FormatException("Invalid or missing 'id'")),
        sensor:
            json["sensor"]?.toString() ??
            (throw const FormatException("Missing 'sensor'")),
        module:
            json["module"]?.toString() ??
            (throw const FormatException("Missing 'module'")),
        section:
            json["section"]?.toString() ??
            (throw const FormatException("Missing 'section'")),
        date:
            DateTime.tryParse(json["date"] ?? "") ??
            (throw const FormatException("Invalid or missing 'date'")),
        value:
            double.tryParse(json["value"]?.toString() ?? "") ??
            (throw const FormatException("Invalid or missing 'value'")),
      );
    } catch (e) {
      throw FormatException("Error parsing Record from JSON: $e");
    }
  }

  final int id;
  final String sensor;
  final String module;
  final String section;
  final DateTime date;
  final double value;

  Record copyWith({
    int? id,
    String? sensor,
    String? module,
    String? section,
    DateTime? date,
    double? value,
  }) => Record(
    id: id ?? this.id,
    sensor: sensor ?? this.sensor,
    module: module ?? this.module,
    section: section ?? this.section,
    date: date ?? this.date,
    value: value ?? this.value,
  );
}
