export const filterNavItems = (navItems, can) => {
    return navItems
        .map(item => {
            // Process children first
            const filteredChildren = item.children
                ? filterNavItems(item.children, can)
                : undefined
            // Check item's permission
            const hasAccess = item.permission ? can(item.permission) : true
            // Only keep the item if:
            // 1. Has required permission (or none required)
            // 2. Has valid children (if parent item)
            const keep = hasAccess && (!item.children || filteredChildren.length > 0)
            return keep ? { ...item, children: filteredChildren } : null
        })
        .filter(Boolean) // Remove null entries
}