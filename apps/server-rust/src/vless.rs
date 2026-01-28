use anyhow::{bail, Result};
use std::net::{Ipv4Addr, Ipv6Addr};
use uuid::Uuid;

pub const VLESS_VERSION: u8 = 0;
pub const CMD_TCP: u8 = 1;
pub const CMD_UDP: u8 = 2;

const ATYP_IPV4: u8 = 1;
const ATYP_DOMAIN: u8 = 2;
const ATYP_IPV6: u8 = 3;

pub fn parse_request(data: &[u8], expected_uuid: Uuid) -> Result<(String, u8, Vec<u8>)> {
    if data.len() < 24 {
        bail!("Data too short: {}", data.len());
    }

    // Version
    let version = data[0];
    if version != VLESS_VERSION {
        bail!("Unsupported version: {}", version);
    }

    // UUID
    let uuid_bytes = &data[1..17];
    let req_uuid = Uuid::from_slice(uuid_bytes)?;
    if req_uuid != expected_uuid {
        bail!("UUID mismatch");
    }

    // Addon length
    let addon_len = data[17] as usize;
    let mut offset = 18 + addon_len;

    if data.len() < offset + 4 {
        bail!("Data too short for command");
    }

    // Command
    let command = data[offset];
    offset += 1;

    // Port (big-endian)
    let port = u16::from_be_bytes([data[offset], data[offset + 1]]);
    offset += 2;

    // Address type
    let atyp = data[offset];
    offset += 1;

    // Parse address
    let host = match atyp {
        ATYP_IPV4 => {
            if data.len() < offset + 4 {
                bail!("Data too short for IPv4");
            }
            let ip = Ipv4Addr::new(data[offset], data[offset + 1], data[offset + 2], data[offset + 3]);
            offset += 4;
            ip.to_string()
        }
        ATYP_DOMAIN => {
            if data.len() < offset + 1 {
                bail!("Data too short for domain length");
            }
            let domain_len = data[offset] as usize;
            offset += 1;
            if data.len() < offset + domain_len {
                bail!("Data too short for domain");
            }
            let domain = String::from_utf8(data[offset..offset + domain_len].to_vec())?;
            offset += domain_len;
            domain
        }
        ATYP_IPV6 => {
            if data.len() < offset + 16 {
                bail!("Data too short for IPv6");
            }
            let mut ip_bytes = [0u8; 16];
            ip_bytes.copy_from_slice(&data[offset..offset + 16]);
            let ip = Ipv6Addr::from(ip_bytes);
            offset += 16;
            ip.to_string()
        }
        _ => bail!("Unsupported address type: {}", atyp),
    };

    let addr = format!("{}:{}", host, port);

    // Remaining data as payload
    let payload = if offset < data.len() {
        data[offset..].to_vec()
    } else {
        Vec::new()
    };

    Ok((addr, command, payload))
}

pub fn build_response() -> Vec<u8> {
    vec![VLESS_VERSION, 0] // version + addon length (0)
}
