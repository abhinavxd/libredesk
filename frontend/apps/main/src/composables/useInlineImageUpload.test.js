import { describe, test, expect, vi } from 'vitest'

vi.mock('./useFileUpload', () => ({
    useFileUpload: () => ({
        upload: vi.fn().mockResolvedValue({ url: '/uploads/abc' })
    })
}))

const { useInlineImageUpload, hasInlineImage, hasPendingInlineUpload } =
    await import('./useInlineImageUpload')

describe('hasInlineImage', () => {
    test('matches inline-image class', () => {
        expect(hasInlineImage('<img class="inline-image" src="/x">')).toBe(true)
    })

    test('rejects unrelated img tags', () => {
        expect(hasInlineImage('<img src="/x" class="something-else">')).toBe(false)
    })

    test('handles falsy input', () => {
        expect(hasInlineImage(null)).toBe(false)
        expect(hasInlineImage(undefined)).toBe(false)
        expect(hasInlineImage('')).toBe(false)
    })
})

describe('hasPendingInlineUpload', () => {
    test('matches data-upload-placeholder', () => {
        expect(hasPendingInlineUpload('<span data-upload-placeholder=""></span>')).toBe(true)
    })

    test('rejects content without marker', () => {
        expect(hasPendingInlineUpload('<p>hello</p>')).toBe(false)
        expect(hasPendingInlineUpload('<img class="inline-image" src="/x">')).toBe(false)
    })

    test('handles falsy input', () => {
        expect(hasPendingInlineUpload(null)).toBe(false)
        expect(hasPendingInlineUpload('')).toBe(false)
    })
})

const makeFile = (name, type) => new File(['x'], name, { type })

const makeEditor = (initialImages = []) => {
    const nodes = initialImages.map((attrs, i) => ({
        type: { name: 'image' },
        attrs,
        nodeSize: 1,
        __pos: i
    }))
    const insertContent = vi.fn()
    return {
        editor: {
            state: {
                doc: {
                    descendants: (cb) => nodes.forEach((n) => cb(n, n.__pos))
                },
                get tr() {
                    return { setNodeMarkup: vi.fn(), delete: vi.fn() }
                }
            },
            view: { dispatch: vi.fn() },
            chain: () => ({
                focus: () => ({
                    insertContent: (n) => {
                        insertContent(n)
                        return { run: vi.fn() }
                    }
                })
            })
        },
        insertContent
    }
}

const makeClipboardEvent = ({ files = [], items = [], types = [] }) => ({
    clipboardData: {
        files: files.length > 0 ? files : null,
        items,
        types
    },
    preventDefault: vi.fn()
})

describe('useInlineImageUpload', () => {
    test('returns handlePaste and handleDrop', () => {
        const { editor } = makeEditor()
        const hooks = useInlineImageUpload({ getEditor: () => editor })
        expect(typeof hooks.handlePaste).toBe('function')
        expect(typeof hooks.handleDrop).toBe('function')
    })

    test('handlePaste returns false when clipboardData is null', () => {
        const { editor } = makeEditor()
        const { handlePaste } = useInlineImageUpload({ getEditor: () => editor })
        expect(handlePaste({}, { clipboardData: null })).toBe(false)
    })

    test('handlePaste bails on text/html paste so ProseMirror handles rich content', () => {
        const { editor, insertContent } = makeEditor()
        const onOtherFiles = vi.fn()
        const { handlePaste } = useInlineImageUpload({
            getEditor: () => editor,
            onOtherFiles
        })
        const event = makeClipboardEvent({ types: ['text/html', 'text/plain'] })
        expect(handlePaste({}, event)).toBe(false)
        expect(insertContent).not.toHaveBeenCalled()
        expect(onOtherFiles).not.toHaveBeenCalled()
    })

    test('handlePaste with text/html AND an image item does not intercept the image', () => {
        const { editor, insertContent } = makeEditor()
        const onOtherFiles = vi.fn()
        const { handlePaste } = useInlineImageUpload({
            getEditor: () => editor,
            onOtherFiles
        })
        const png = makeFile('embedded.png', 'image/png')
        const event = {
            clipboardData: {
                files: null,
                items: [
                    { kind: 'string', type: 'text/html' },
                    { kind: 'string', type: 'text/plain' },
                    { kind: 'file', type: 'image/png', getAsFile: () => png }
                ],
                types: ['text/html', 'text/plain', 'image/png']
            },
            preventDefault: vi.fn()
        }
        expect(handlePaste({}, event)).toBe(false)
        expect(insertContent).not.toHaveBeenCalled()
        expect(onOtherFiles).not.toHaveBeenCalled()
        expect(event.preventDefault).not.toHaveBeenCalled()
    })

    test('handlePaste inserts placeholder for image file', () => {
        const { editor, insertContent } = makeEditor()
        const onOtherFiles = vi.fn()
        const { handlePaste } = useInlineImageUpload({
            getEditor: () => editor,
            onOtherFiles
        })
        const png = makeFile('a.png', 'image/png')
        expect(handlePaste({}, makeClipboardEvent({ files: [png] }))).toBe(true)
        expect(insertContent).toHaveBeenCalledTimes(1)
        const nodes = insertContent.mock.calls[0][0]
        expect(nodes).toHaveLength(1)
        expect(nodes[0]).toMatchObject({
            type: 'image',
            attrs: { uploading: true, uploadName: 'a.png' }
        })
        expect(onOtherFiles).not.toHaveBeenCalled()
    })

    test('handlePaste routes SVG to onOtherFiles (not inline)', () => {
        const { editor, insertContent } = makeEditor()
        const onOtherFiles = vi.fn()
        const { handlePaste } = useInlineImageUpload({
            getEditor: () => editor,
            onOtherFiles
        })
        const svg = makeFile('icon.svg', 'image/svg+xml')
        expect(handlePaste({}, makeClipboardEvent({ files: [svg] }))).toBe(true)
        expect(insertContent).not.toHaveBeenCalled()
        expect(onOtherFiles).toHaveBeenCalledWith([svg])
    })

    test('handlePaste routes non-image file to onOtherFiles', () => {
        const { editor, insertContent } = makeEditor()
        const onOtherFiles = vi.fn()
        const { handlePaste } = useInlineImageUpload({
            getEditor: () => editor,
            onOtherFiles
        })
        const csv = makeFile('a.csv', 'text/csv')
        expect(handlePaste({}, makeClipboardEvent({ files: [csv] }))).toBe(true)
        expect(insertContent).not.toHaveBeenCalled()
        expect(onOtherFiles).toHaveBeenCalledWith([csv])
    })

    test('handlePaste partitions mixed image and non-image files', () => {
        const { editor, insertContent } = makeEditor()
        const onOtherFiles = vi.fn()
        const { handlePaste } = useInlineImageUpload({
            getEditor: () => editor,
            onOtherFiles
        })
        const event = makeClipboardEvent({
            files: [
                makeFile('a.png', 'image/png'),
                makeFile('b.csv', 'text/csv'),
                makeFile('c.jpg', 'image/jpeg')
            ]
        })
        expect(handlePaste({}, event)).toBe(true)
        expect(insertContent.mock.calls[0][0]).toHaveLength(2)
        expect(onOtherFiles).toHaveBeenCalledWith([
            expect.objectContaining({ name: 'b.csv' })
        ])
    })

    test('handlePaste items branch picks up non-image file items', () => {
        const { editor, insertContent } = makeEditor()
        const onOtherFiles = vi.fn()
        const { handlePaste } = useInlineImageUpload({
            getEditor: () => editor,
            onOtherFiles
        })
        const csv = makeFile('a.csv', 'text/csv')
        const event = {
            clipboardData: {
                files: null,
                items: [{ kind: 'file', type: 'text/csv', getAsFile: () => csv }],
                types: ['Files']
            },
            preventDefault: vi.fn()
        }
        expect(handlePaste({}, event)).toBe(true)
        expect(insertContent).not.toHaveBeenCalled()
        expect(onOtherFiles).toHaveBeenCalledWith([csv])
    })

    test('isInlineEnabled=false routes images to onOtherFiles', () => {
        const { editor, insertContent } = makeEditor()
        const onOtherFiles = vi.fn()
        const { handlePaste } = useInlineImageUpload({
            getEditor: () => editor,
            isInlineEnabled: () => false,
            onOtherFiles
        })
        const png = makeFile('a.png', 'image/png')
        expect(handlePaste({}, makeClipboardEvent({ files: [png] }))).toBe(true)
        expect(insertContent).not.toHaveBeenCalled()
        expect(onOtherFiles).toHaveBeenCalledWith([png])
    })

    test('maxInlineImages cap truncates inline insertions', () => {
        const { editor, insertContent } = makeEditor([
            { uploading: false },
            { uploading: false }
        ])
        const { handlePaste } = useInlineImageUpload({
            getEditor: () => editor,
            maxInlineImages: 3
        })
        const event = makeClipboardEvent({
            files: [
                makeFile('1.png', 'image/png'),
                makeFile('2.png', 'image/png'),
                makeFile('3.png', 'image/png'),
                makeFile('4.png', 'image/png')
            ]
        })
        expect(handlePaste({}, event)).toBe(true)
        expect(insertContent.mock.calls[0][0]).toHaveLength(1)
    })

    test('handleDrop dispatches files like a paste', () => {
        const { editor, insertContent } = makeEditor()
        const onOtherFiles = vi.fn()
        const { handleDrop } = useInlineImageUpload({
            getEditor: () => editor,
            onOtherFiles
        })
        const event = {
            dataTransfer: {
                files: [makeFile('a.png', 'image/png'), makeFile('b.pdf', 'application/pdf')]
            },
            preventDefault: vi.fn()
        }
        expect(handleDrop({}, event)).toBe(true)
        expect(insertContent.mock.calls[0][0]).toHaveLength(1)
        expect(onOtherFiles).toHaveBeenCalledWith([
            expect.objectContaining({ name: 'b.pdf' })
        ])
    })

    test('handleDrop returns false with no files', () => {
        const { editor } = makeEditor()
        const { handleDrop } = useInlineImageUpload({ getEditor: () => editor })
        expect(handleDrop({}, { dataTransfer: { files: [] } })).toBe(false)
    })
})
