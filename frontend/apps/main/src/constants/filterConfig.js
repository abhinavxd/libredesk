export const FIELD_TYPE = {
    SELECT: 'select',
    TAG: 'tag',
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
    LESS_THAN: 'less than'
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
    DATE: [
        OPERATOR.EQUALS,
        OPERATOR.NOT_EQUALS,
        OPERATOR.SET,
        OPERATOR.NOT_SET,
        OPERATOR.GREATER_THAN,
        OPERATOR.LESS_THAN
    ],
    NUMBER: [OPERATOR.EQUALS, OPERATOR.NOT_EQUALS, OPERATOR.GREATER_THAN, OPERATOR.LESS_THAN],
}
