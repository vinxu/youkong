import type { VercelRequest, VercelResponse } from '@vercel/node'
import { createHash } from 'crypto'
import { createRequire } from 'module'

const require = createRequire(import.meta.url)
const COS = require('cos-nodejs-sdk-v5')

const GIPHY_API_KEY = process.env.GIPHY_API_KEY || ''
const GIPHY_SEARCH_URL = 'https://api.giphy.com/v1/gifs/search'

// COS config
const COS_SECRET_ID = process.env.TENCENT_COS_SECRET_ID || ''
const COS_SECRET_KEY = process.env.TENCENT_COS_SECRET_KEY || ''
const COS_BUCKET = process.env.TENCENT_COS_BUCKET || ''
const COS_REGION = process.env.TENCENT_COS_REGION || ''

function md5(str: string): string {
  return createHash('md5').update(str).digest('hex')
}

function cosEnabled(): boolean {
  return !!(COS_SECRET_ID && COS_SECRET_KEY && COS_BUCKET && COS_REGION)
}

let cosClient: any = null
function getCosClient(): any {
  if (!cosClient && cosEnabled()) {
    cosClient = new COS({
      SecretId: COS_SECRET_ID,
      SecretKey: COS_SECRET_KEY,
    })
  }
  return cosClient
}

async function uploadToCOS(gifUrl: string, buffer: Buffer): Promise<string> {
  const cos = getCosClient()
  if (!cos) throw new Error('COS not configured')

  const urlHash = md5(gifUrl)
  const key = `gifs/${urlHash}.gif`

  return new Promise((resolve, reject) => {
    cos.putObject(
      {
        Bucket: COS_BUCKET,
        Region: COS_REGION,
        Key: key,
        Body: buffer,
        ContentType: 'image/gif',
        ACL: 'public-read',
      },
      (err: any) => {
        if (err) {
          reject(err)
        } else {
          resolve(
            `https://${COS_BUCKET}.cos.${COS_REGION}.myqcloud.com/${key}`
          )
        }
      }
    )
  })
}

export default async function handler(req: VercelRequest, res: VercelResponse) {
  // CORS
  res.setHeader('Access-Control-Allow-Origin', '*')
  res.setHeader('Access-Control-Allow-Methods', 'GET')

  if (req.method === 'OPTIONS') {
    return res.status(200).end()
  }

  const query = req.query.q as string
  if (!query) {
    return res.status(400).json({ error: 'missing q parameter' })
  }

  // Optional auth
  const token = req.query.token as string
  const expectedToken = process.env.API_TOKEN || ''
  if (expectedToken && token !== expectedToken) {
    return res.status(401).json({ error: 'unauthorized' })
  }

  try {
    // 1. Search Giphy
    const params = new URLSearchParams({
      api_key: GIPHY_API_KEY,
      q: query,
      limit: '25',
      rating: 'g',
      lang: 'en',
    })

    const resp = await fetch(`${GIPHY_SEARCH_URL}?${params}`)
    if (!resp.ok) {
      return res
        .status(resp.status)
        .json({ error: `Giphy API error: ${resp.status}` })
    }

    const data = await resp.json()
    if (!data.data || data.data.length === 0) {
      return res.status(200).json({ result: null })
    }

    // 2. Deterministic pick: use seed if provided, otherwise fallback to query hash
    const seed = req.query.seed as string
    const hashSource = seed ? seed : query.toLowerCase().trim()
    const hashValue = md5(hashSource)
    const hashNum = parseInt(hashValue.substring(0, 8), 16)
    const idx = hashNum % data.data.length
    const gif = data.data[idx]
    const images = gif.images || {}
    const fixedHeightUrl: string = images.fixed_height?.url || ''

    // 3. Try download + COS upload
    let cosUrl = ''
    if (fixedHeightUrl && cosEnabled()) {
      try {
        const dlResp = await fetch(fixedHeightUrl)
        if (dlResp.ok) {
          const buffer = Buffer.from(await dlResp.arrayBuffer())
          cosUrl = await uploadToCOS(fixedHeightUrl, buffer)
        }
      } catch (cosErr: any) {
        console.error('COS upload failed, returning CDN URL:', cosErr.message)
      }
    }

    const result = {
      cos_url: cosUrl,
      cdn_url: fixedHeightUrl,
      width: parseInt(images.fixed_height?.width || '0'),
      height: parseInt(images.fixed_height?.height || '0'),
    }

    return res.status(200).json({ result })
  } catch (err: any) {
    return res.status(500).json({ error: err.message || 'internal error' })
  }
}
