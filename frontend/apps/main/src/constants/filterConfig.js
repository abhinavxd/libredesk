export const LOGIC = {
    AND: 'AND',
    OR: 'OR'
}

// Mirrors dbutil.MaxFilterGroups on the backend.
export const MAX_FILTER_GROUPS = 10

export const FIELD_TYPE = {
    SELECT: 'select',
    TAG: 'tag',
    MULTI_SELECT: 'multi-select',
    TEXT: 'text',
    NUMBER: 'number',
    RICHTEXT: 'richtext',
    BOOLEAN: 'boolean',
    DATE: 'date',
}

export const OPERATOR = {
    EQUALS: 'equals',
    NOT_EQUALS: 'not equals',
    SET: 'set',
    NOT_SET: 'not set',
    CONTAINS: 'contains',
    NOT_CONTAINS: 'not contains',
    GREATER_THAN: 'greater than',
    LESS_THAN: 'less than',
    BETWEEN: 'between'
}

// operatorLabel returns a clearer display label for operators whose meaning is ambiguous with
// multiple values (contains = matches ANY of the values). Other operators display as-is.
const OPERATOR_LABEL_KEYS = {
    [OPERATOR.CONTAINS]: 'filter.containsAnyOf',
    [OPERATOR.NOT_CONTAINS]: 'filter.containsNoneOf'
}
export const operatorLabel = (op, t) => {
    const key = OPERATOR_LABEL_KEYS[op]
    return key ? t(key) : op
}

export const FIELD_OPERATORS = {
    SELECT: [OPERATOR.EQUALS, OPERATOR.NOT_EQUALS, OPERATOR.SET, OPERATOR.NOT_SET],
    BOOLEAN: [OPERATOR.EQUALS, OPERATOR.NOT_EQUALS],
    TEXT: [
        OPERATOR.EQUALS,
        OPERATOR.NOT_EQUALS,
        OPERATOR.SET,
        OPERATOR.NOT_SET,
        OPERATOR.CONTAINS,
        OPERATOR.NOT_CONTAINS
    ],
    // For text columns that do not support partial matching, only allow exact match operators.
    TEXT_EXACT: [OPERATOR.EQUALS, OPERATOR.NOT_EQUALS, OPERATOR.SET, OPERATOR.NOT_SET],
    DATE: [
        OPERATOR.EQUALS,
        OPERATOR.NOT_EQUALS,
        OPERATOR.SET,
        OPERATOR.NOT_SET,
        OPERATOR.GREATER_THAN,
        OPERATOR.LESS_THAN,
        OPERATOR.BETWEEN
    ],
    NUMBER: [OPERATOR.EQUALS, OPERATOR.NOT_EQUALS, OPERATOR.GREATER_THAN, OPERATOR.LESS_THAN],
    MULTI_SELECT: [OPERATOR.CONTAINS, OPERATOR.NOT_CONTAINS, OPERATOR.SET, OPERATOR.NOT_SET]
}
