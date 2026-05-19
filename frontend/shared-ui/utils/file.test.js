import { describe, test, expect } from 'vitest'
import { downloadUrl, getThumbFilepath } from './file'

describe('downloadUrl', () => {
    test('returns falsy input unchanged', () => {
        expect(downloadUrl('')).toBe('')
        expect(downloadUrl(null)).toBe(null)
        expect(downloadUrl(undefined)).toBe(undefined)
    })

    test('rewrites relative fs upload path', () => {
        expect(downloadUrl('/uploads/9a4f0a03-7b36-4e05-aacd-4eb04947a79d'))
            .toBe('/uploads/9a4f0a03-7b36-4e05-aacd-4eb04947a79d?download=1')
    })

    test('strips signature params from signed fs url', () => {
        const signed = 'https://example.com/uploads/9a4f0a03-7b36-4e05-aacd-4eb04947a79d?sig=abc&exp=1700000000'
        expect(downloadUrl(signed))
            .toBe('/uploads/9a4f0a03-7b36-4e05-aacd-4eb04947a79d?download=1')
    })

    test('extracts uuid from public s3 url', () => {
        const s3 = 'https://s3.ap-south-1.amazonaws.com/example-bucket/9a4f0a03-7b36-4e05-aacd-4eb04947a79d'
        expect(downloadUrl(s3))
            .toBe('/uploads/9a4f0a03-7b36-4e05-aacd-4eb04947a79d?download=1')
    })

    test('extracts uuid from presigned s3 url with query params', () => {
        const presigned = 'https://s3.ap-south-1.amazonaws.com/example-bucket/9a4f0a03-7b36-4e05-aacd-4eb04947a79d?X-Amz-Algorithm=AWS4-HMAC-SHA256&X-Amz-Expires=300&response-content-disposition=inline%3B%20filename%3D%22photo.jpg%22&X-Amz-Signature=deadbeef'
        expect(downloadUrl(presigned))
            .toBe('/uploads/9a4f0a03-7b36-4e05-aacd-4eb04947a79d?download=1')
    })

    test('extracts uuid from virtual-hosted s3 url', () => {
        const s3 = 'https://example-bucket.s3.ap-south-1.amazonaws.com/9a4f0a03-7b36-4e05-aacd-4eb04947a79d?X-Amz-Signature=deadbeef'
        expect(downloadUrl(s3))
            .toBe('/uploads/9a4f0a03-7b36-4e05-aacd-4eb04947a79d?download=1')
    })

    test('does not double-append download param when already present', () => {
        const already = '/uploads/9a4f0a03-7b36-4e05-aacd-4eb04947a79d?download=1'
        expect(downloadUrl(already))
            .toBe('/uploads/9a4f0a03-7b36-4e05-aacd-4eb04947a79d?download=1')
    })

    test('handles unencoded slash inside query param', () => {
        const url = 'https://s3.example.com/bucket/9a4f0a03-7b36-4e05-aacd-4eb04947a79d?response-content-disposition=inline;filename="a/b.jpg"'
        expect(downloadUrl(url))
            .toBe('/uploads/9a4f0a03-7b36-4e05-aacd-4eb04947a79d?download=1')
    })

    test('returns original url when no uuid found', () => {
        expect(downloadUrl('https://example.com/some/path')).toBe('https://example.com/some/path')
    })

    test('returns first uuid when multiple are present', () => {
        const url = 'https://s3.example.com/11111111-1111-4111-8111-111111111111/9a4f0a03-7b36-4e05-aacd-4eb04947a79d'
        expect(downloadUrl(url))
            .toBe('/uploads/11111111-1111-4111-8111-111111111111?download=1')
    })

    test('matches uppercase uuid', () => {
        const url = 'https://s3.example.com/bucket/9A4F0A03-7B36-4E05-AACD-4EB04947A79D'
        expect(downloadUrl(url))
            .toBe('/uploads/9A4F0A03-7B36-4E05-AACD-4EB04947A79D?download=1')
    })
})

describe('getThumbFilepath', () => {
    test('returns falsy input unchanged', () => {
        expect(getThumbFilepath('')).toBe('')
        expect(getThumbFilepath(null)).toBe(null)
        expect(getThumbFilepath(undefined)).toBe(undefined)
    })

    test('builds thumb path from relative fs upload path', () => {
        expect(getThumbFilepath('/uploads/9a4f0a03-7b36-4e05-aacd-4eb04947a79d'))
            .toBe('/uploads/thumb_9a4f0a03-7b36-4e05-aacd-4eb04947a79d')
    })

    test('preserves signature params from signed fs url', () => {
        const signed = 'https://example.com/uploads/9a4f0a03-7b36-4e05-aacd-4eb04947a79d?sig=abc&exp=1700000000'
        expect(getThumbFilepath(signed))
            .toBe('/uploads/thumb_9a4f0a03-7b36-4e05-aacd-4eb04947a79d?sig=abc&exp=1700000000')
    })

    test('preserves query string from presigned s3 url', () => {
        const presigned = 'https://s3.ap-south-1.amazonaws.com/example-bucket/9a4f0a03-7b36-4e05-aacd-4eb04947a79d?X-Amz-Algorithm=AWS4-HMAC-SHA256&X-Amz-Signature=deadbeef'
        expect(getThumbFilepath(presigned))
            .toBe('/uploads/thumb_9a4f0a03-7b36-4e05-aacd-4eb04947a79d?X-Amz-Algorithm=AWS4-HMAC-SHA256&X-Amz-Signature=deadbeef')
    })

    test('returns original url when no uuid found', () => {
        expect(getThumbFilepath('https://example.com/some/path'))
            .toBe('https://example.com/some/path')
    })
})
