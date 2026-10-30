
import { defineMessage as msg } from '@lingui/core/macro'
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
    WEBHOOK: 'webhook',
    RECIPIENTS: 'recipients',
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
    BETWEEN: 'between',
    STARTS_WITH: 'starts with'
}

// operatorLabel returns a clearer display label for operators whose meaning is ambiguous with
// multiple values (contains = matches ANY of the values). Other operators display as-is.
const OPERATOR_MESSAGES = {
    [OPERATOR.CONTAINS]: msg({ message: 'contains any of' }),
    [OPERATOR.NOT_CONTAINS]: msg({ message: 'contains none of' })
}
export const operatorLabel = (op, i18n) => {
    const message = OPERATOR_MESSAGES[op]
    return message ? i18n._(message) : op
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
    // "starts with" is only implemented by the automation evaluator, not the SQL filter builder.
    TEXT_AUTOMATION: [
        OPERATOR.EQUALS,
        OPERATOR.NOT_EQUALS,
        OPERATOR.SET,
        OPERATOR.NOT_SET,
        OPERATOR.CONTAINS,
        OPERATOR.NOT_CONTAINS,
        OPERATOR.STARTS_WITH
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
    // "between" is only implemented by the SQL filter builder, not the automation evaluator.
    DATE_AUTOMATION: [
        OPERATOR.EQUALS,
        OPERATOR.NOT_EQUALS,
        OPERATOR.SET,
        OPERATOR.NOT_SET,
        OPERATOR.GREATER_THAN,
        OPERATOR.LESS_THAN
    ],
    NUMBER: [OPERATOR.EQUALS, OPERATOR.NOT_EQUALS, OPERATOR.GREATER_THAN, OPERATOR.LESS_THAN],
    MULTI_SELECT: [OPERATOR.CONTAINS, OPERATOR.NOT_CONTAINS, OPERATOR.SET, OPERATOR.NOT_SET]
}
